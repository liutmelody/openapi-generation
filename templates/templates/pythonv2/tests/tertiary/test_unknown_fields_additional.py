import json
from typing import List

import pydantic
import pytest

from openapi import SDK, utils
from openapi.models import errors, shared
from .common_helpers import HTTPBIN_URL
from .unknown_fields_cases import UNKNOWN_FIELD_CASES


@pytest.mark.parametrize("fields, known_fields", UNKNOWN_FIELD_CASES)
def test_to_dict_drops_unknown_fields(fields, known_fields):
    model = shared.FieldEnvelopeInput.model_validate({"label": "root", **fields})
    expected = {"label": "root", **known_fields}
    assert model.to_dict(mode="json") == expected
    assert json.loads(model.to_json()) == expected
    assert shared.FieldEnvelopeInput.model_validate(model.to_dict()).to_dict(mode="json") == expected


def test_flattened_request_drops_nested_unknown_fields():
    with SDK(server_url=HTTPBIN_URL) as sdk:
        response = sdk.echo_flattened_unknown_fields(label="root", child={"displayName": "child", "future": True})
    assert response.raw_response.json()["json"] == {"label": "root", "child": {"displayName": "child"}}
    assert response.object.json_.to_dict() == {"label": "root", "child": {"displayName": "child"}}
    assert "headers" not in response.object.to_dict()


@pytest.mark.parametrize("options, expected", [
    ({}, {"displayName": "child", "nullable": None}),
    ({"use_api_names": False}, {"display_name": "child", "nullable": None}),
    ({"exclude_unset": False}, {"displayName": "child", "nullable": None}),
    ({"exclude_defaults": True}, {"displayName": "child", "nullable": None}),
    ({"exclude_none": True}, {"displayName": "child"}),
])
def test_model_serialization_methods(options, expected):
    model = shared.FieldNode.model_validate({"displayName": "child", "nullable": None})
    assert model.to_dict(**options) == expected
    assert model.to_dict(mode="json", **options) == expected
    assert json.loads(model.to_json(**options)) == expected


def test_exclude_none_keeps_explicit_nulls_on_the_wire():
    model = shared.FieldNode(display_name="child", nullable=None)
    assert model.to_dict(exclude_none=True) == {"displayName": "child"}
    assert json.loads(utils.marshal_json(model, shared.FieldNode)) == {"displayName": "child", "nullable": None}


def test_excluded_fields_stay_out_of_serialization_methods():
    model = shared.DefaultsAndConsts(normal_field="x")
    assert "defaultStrNullable" not in model.to_dict()
    assert model.to_dict(exclude_unset=False)["defaultStrNullable"] is None
    assert shared.DefaultsAndConsts(normal_field="x", default_str_nullable=None).to_dict()["defaultStrNullable"] is None
    assert json.loads(utils.marshal_json(model, shared.DefaultsAndConsts))["defaultStrNullable"] is None
    node = shared.FieldNode(display_name="child", nullable=None)
    assert node.model_dump(by_alias=True, exclude={"display_name"}) == {"nullable": None}
    assert node.model_dump(by_alias=True, exclude_none=True) == {"displayName": "child", "nullable": None}


def test_model_serialization_methods_field_names():
    model = shared.FieldEnvelopeInput.model_validate({"label": "root", "child": {"displayName": "child"}, "futureKind": "third", "plainChoice": {"text": "t"}})
    api_names = {"label": "root", "child": {"displayName": "child"}, "futureKind": "third", "plainChoice": {"text": "t"}}
    field_names = {"label": "root", "child": {"display_name": "child"}, "future_kind": "third", "plain_choice": {"text": "t"}}
    assert model.to_dict() == api_names
    assert model.to_dict(use_api_names=False) == field_names
    assert json.loads(model.to_json(use_api_names=False)) == field_names
    assert shared.FieldEnvelopeInput.model_validate(model.to_dict(use_api_names=False)).to_dict() == api_names


def test_default_dump_keeps_nested_api_names():
    model = shared.FieldEnvelopeInput.model_validate({"label": "root", "child": {"displayName": "child", "child": {"displayName": "grandchild"}}, "children": [{"displayName": "item"}]})
    expected = {"label": "root", "child": {"displayName": "child", "child": {"displayName": "grandchild"}}, "children": [{"displayName": "item"}]}
    assert model.model_dump() == expected
    assert json.loads(model.model_dump_json()) == expected
    assert pydantic.TypeAdapter(List[shared.FieldNode]).dump_python([shared.FieldNode(display_name="item")]) == [{"displayName": "item"}]


def test_helper_names_reserved_only_on_models():
    assert set(shared.MethodNames.model_fields) == {"to_dict_", "to_json_"}
    model = shared.MethodNames.model_validate({"toDict": "d", "toJson": "j"})
    assert model.to_dict() == {"toDict": "d", "toJson": "j"}
    with SDK(server_url=HTTPBIN_URL) as sdk:
        response = sdk.to_json.echo_grouped_fields()
    assert response.status_code == 200


def test_model_serialization_methods_keep_consts():
    model = shared.FieldEnvelopeInput(label="root", choice=shared.TextChoice(text="t"), constant=shared.ConstFields())
    expected = {"label": "root", "choice": {"kind": "text", "text": "t"}, "constant": {"kind": "fixed"}}
    wire = {"label": "root", "choice": {"kind": "text", "text": "t"}, "constant": {"kind": "fixed", "priority": 1}}
    assert json.loads(utils.marshal_json(model, shared.FieldEnvelopeInput)) == wire
    assert model.to_dict() == expected
    assert model.to_dict(exclude_unset=False) == wire
    assert json.loads(model.to_json()) == expected
    assert shared.FieldEnvelopeInput.model_validate(model.to_dict()).to_dict() == expected
    assert shared.FieldEnvelopeInput.model_validate(wire).to_dict() == wire
    assert shared.ConstFields(priority=1).to_dict() == {"kind": "fixed", "priority": 1}
    assert shared.ConstFields(priority=1).to_dict(exclude_defaults=True) == {}
    assert shared.ConstFields(priority=2).to_dict(exclude_defaults=True) == {"priority": 2}


def test_model_serialization_methods_keep_error_data_consts():
    data = errors.EchoUnknownFieldsResponseBodyData(message="bad")
    assert data.model_dump(by_alias=True) == {"message": "bad", "kind": "invalid", "detail": None}
    assert data.to_dict() == {"message": "bad", "kind": "invalid"}
    assert json.loads(data.to_json()) == {"message": "bad", "kind": "invalid"}
    assert data.to_dict(exclude_defaults=True) == {"message": "bad"}


def test_optional_null_consts_keep_wire_output():
    model = shared.ObjectWithNullEnums()
    assert json.loads(utils.marshal_json(model, shared.ObjectWithNullEnums)) == {"nullEnum": None}
    assert model.to_dict() == {"nullEnum": None}
    explicit = shared.ObjectWithNullEnums(opt_null_enum=None)
    assert json.loads(utils.marshal_json(explicit, shared.ObjectWithNullEnums)) == {"nullEnum": None, "optNullEnum": None}
    assert explicit.to_dict() == {"nullEnum": None, "optNullEnum": None}
