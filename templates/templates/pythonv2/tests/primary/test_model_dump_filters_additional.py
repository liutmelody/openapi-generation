import json
from typing import List, Optional

from pydantic import TypeAdapter

from openapi import utils
from openapi.models import errors, shared
from openapi.types import Nullable


def filtered_fields(display_name="name", **kwargs):
    return shared.FilteredFields(
        display_name=display_name, required_nullable=None, **kwargs
    )


class TestModelDumpFilters:
    def test_exclude_leaves_required_field_out(self):
        model = filtered_fields()

        assert "displayName" not in model.model_dump(exclude={"display_name"})
        assert "displayName" not in model.model_dump(
            by_alias=True, exclude={"display_name"}
        )
        assert "displayName" not in json.loads(
            model.model_dump_json(exclude={"display_name"})
        )

    def test_include_leaves_other_fields_out(self):
        model = filtered_fields(optional_nullable="x")

        assert model.model_dump(include={"optional_nullable"}) == {
            "optionalNullable": "x"
        }
        assert model.model_dump(by_alias=True, include={"display_name"}) == {
            "displayName": "name"
        }

    def test_exclude_unset_leaves_unset_fields_out(self):
        model = filtered_fields()

        assert model.model_dump(exclude_unset=True) == {
            "displayName": "name",
            "requiredNullable": None,
        }

    def test_exclude_defaults_leaves_default_fields_out(self):
        model = filtered_fields(with_default="other")

        assert model.model_dump(exclude_defaults=True) == {
            "displayName": "name",
            "requiredNullable": None,
            "withDefault": "other",
        }

    def test_exclude_none_still_writes_back_excluded_required_field(self):
        model = filtered_fields()

        dumped = model.model_dump(exclude={"display_name"}, exclude_none=True)

        assert dumped["displayName"] is None

    def test_nested_exclude(self):
        model = filtered_fields(
            child=filtered_fields("child"), children=[filtered_fields("item")]
        )

        dumped = model.model_dump(
            exclude={
                "child": {"display_name"},
                "children": {"__all__": {"display_name"}},
            }
        )

        assert dumped["displayName"] == "name"
        assert "displayName" not in dumped["child"]
        assert "displayName" not in dumped["children"][0]


class TestNestedDefaultDumps:
    def test_nested_models_keep_api_names(self):
        model = filtered_fields(
            child=filtered_fields("child"), children=[filtered_fields("item")]
        )

        for dumped in (model.model_dump(), json.loads(model.model_dump_json())):
            assert dumped["child"]["displayName"] == "child"
            assert dumped["child"]["requiredNullable"] is None
            assert dumped["children"][0]["displayName"] == "item"

    def test_model_lists_keep_api_names(self):
        adapter = TypeAdapter(List[shared.FilteredFields])

        dumped = adapter.dump_python([filtered_fields("first")])

        assert dumped[0]["displayName"] == "first"
        assert dumped[0]["requiredNullable"] is None


class TestRequestBodies:
    def test_explicit_nulls(self):
        model = filtered_fields(optional_nullable=None)

        assert (
            utils.marshal_json(model, shared.FilteredFields)
            == '{"displayName":"name","requiredNullable":null,"kind":"filtered","nullDefault":null,"optionalNullable":null,"withDefault":"fallback"}'
        )

    def test_required_nullable_null(self):
        model = shared.PaginationCursorNullableLimitNextCursor(
            next_cursor=None, results=["a"]
        )

        assert (
            utils.marshal_json(model, shared.PaginationCursorNullableLimitNextCursor)
            == '{"next_cursor":null,"results":["a"]}'
        )

    def test_nested_nulls_and_api_names(self):
        model = filtered_fields(
            child=filtered_fields("child", optional_nullable=None),
            children=[filtered_fields("item")],
        )

        assert json.loads(utils.marshal_json(model, shared.FilteredFields)) == {
            "displayName": "name",
            "requiredNullable": None,
            "child": {
                "displayName": "child",
                "requiredNullable": None,
                "kind": "filtered",
                "nullDefault": None,
                "optionalNullable": None,
                "withDefault": "fallback",
            },
            "children": [
                {
                    "displayName": "item",
                    "requiredNullable": None,
                    "kind": "filtered",
                    "nullDefault": None,
                    "withDefault": "fallback",
                }
            ],
            "kind": "filtered",
            "nullDefault": None,
            "withDefault": "fallback",
        }

    def test_null_additional_property_omitted(self):
        model = shared.ObjWithAnyAdditionalProperties.model_validate(
            {"normalField": "x", "future": None}
        )

        assert (
            utils.marshal_json(model, shared.ObjWithAnyAdditionalProperties)
            == '{"normalField":"x"}'
        )

    def test_additional_property_under_python_name(self):
        model = shared.ObjWithAnyAdditionalProperties.model_validate(
            {"normalField": None, "normal_field": "extra"}
        )
        other = shared.FilteredFieldsWithAdditionalProperties.model_validate(
            {"displayName": None, "display_name": "extra"}
        )

        assert (
            utils.marshal_json(model, shared.ObjWithAnyAdditionalProperties)
            == '{"normalField":"extra"}'
        )
        assert (
            utils.marshal_json(other, shared.FilteredFieldsWithAdditionalProperties)
            == '{"displayName":"extra"}'
        )

    def test_top_level_none(self):
        assert utils.marshal_json(None, Optional[shared.FilteredFields]) == ""
        assert utils.marshal_json(None, Nullable[shared.FilteredFields]) == "null"


class TestErrorData:
    def test_plain_filtering(self):
        data = errors.FilteredFieldsErrorData()

        assert data.model_dump() == {"detail": None}
        assert data.model_dump(exclude_none=True) == {}
        assert data.model_dump(exclude_unset=True) == {}

    def test_nested_model_honours_exclude(self):
        data = errors.FilteredFieldsErrorData(detail=filtered_fields())

        dumped = data.model_dump(exclude={"detail": {"display_name"}})

        assert "displayName" not in dumped["detail"]
        assert dumped["detail"]["requiredNullable"] is None


class TestEmptyAlias:
    def test_value_kept_under_empty_api_name(self):
        model = shared.BinaryString(file_name="a", content=b"x")

        assert model.model_dump(by_alias=True)[""] == b"x"
        assert model.model_dump()[""] == b"x"
        assert "content" not in model.model_dump(by_alias=True)
        assert "" not in model.model_dump(by_alias=True, exclude={"content"})
