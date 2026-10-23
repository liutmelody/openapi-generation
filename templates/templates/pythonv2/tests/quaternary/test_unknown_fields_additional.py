import json
import warnings

import pydantic
import pytest

from openapi import SDK, AsyncSDK, utils
from openapi.models import shared
from openapi.types import UnrecognizedInt, UnrecognizedStr
from .common_helpers import HTTPBIN_URL, record_test
from .unknown_fields import drop_unexpected_fields
from .unknown_fields_cases import UNKNOWN_FIELD_CASES


@pytest.mark.parametrize("child", [
    {"displayName": "child", "future": True},
    {"displayName": "child", "display_name": "unknown"},
    {"displayName": "child", "child": {"displayName": "grandchild", "future": {"nested": [None, 0]}}},
    {"displayName": "child", "nullable": None, "future": None},
])
def test_flattened_request_keeps_nested_unknown_fields(child):
    record_test("unknown-fields-json-round-trip")
    with SDK(server_url=HTTPBIN_URL) as sdk:
        response = sdk.echo_unknown_fields(label="root", child=child)
    payload = {"label": "root", "child": child}
    assert response.raw_response.json()["json"] == payload
    assert response.object.json_.to_dict() == payload
    assert json.loads(response.object.json_.to_json()) == payload


@pytest.mark.asyncio()
async def test_flattened_request_keeps_nested_unknown_fields_async():
    child = {"displayName": "child", "child": {"displayName": "grandchild", "future": {"nested": [None, 0]}}}
    async with AsyncSDK(server_url=HTTPBIN_URL) as sdk:
        response = await sdk.echo_unknown_fields(label="root", child=child)
    payload = {"label": "root", "child": child}
    assert response.raw_response.json()["json"] == payload
    assert response.object.json_.to_dict() == payload
    assert json.loads(response.object.json_.to_json()) == payload


def test_flattened_request_keeps_unknown_fields_on_nested_models():
    child = shared.FieldNode(display_name="child", future=None)
    with SDK(server_url=HTTPBIN_URL) as sdk:
        response = sdk.echo_flattened_unknown_fields(label="root", child=child)
    assert response.raw_response.json()["json"] == {"label": "root", "child": {"displayName": "child", "future": None}}


def test_flattened_request_has_no_argument_for_top_level_unknown_fields():
    with SDK(server_url=HTTPBIN_URL) as sdk:
        with pytest.raises(TypeError):
            sdk.echo_flattened_unknown_fields(label="root", future=True)  # type: ignore[call-arg]


@pytest.mark.parametrize("fields", [fields for fields, _ in UNKNOWN_FIELD_CASES])
def test_to_dict_round_trips_unknown_fields(fields):
    payload = {"label": "root", **fields}
    model = shared.FieldEnvelopeInput.model_validate(payload)
    assert model.to_dict(mode="json") == payload
    assert json.loads(model.to_json()) == payload
    assert shared.FieldEnvelopeInput.model_validate(model.to_dict()).to_dict(mode="json") == payload


@pytest.mark.parametrize("options, expected", [
    ({}, {"displayName": "child", "nullable": None, "future": None}),
    ({"use_api_names": False}, {"display_name": "child", "nullable": None, "future": None}),
    ({"exclude_unset": False}, {"displayName": "child", "nullable": None, "future": None}),
    ({"exclude_defaults": True}, {"displayName": "child", "nullable": None, "future": None}),
    ({"exclude_none": True}, {"displayName": "child"}),
])
def test_model_serialization_methods(options, expected):
    model = shared.FieldNode.model_validate({"displayName": "child", "nullable": None, "future": None})
    assert model.to_dict(**options) == expected
    assert model.to_dict(mode="json", **options) == expected
    assert json.loads(model.to_json(**options)) == expected


@pytest.mark.parametrize("options", [{"exclude_none": True}, {"exclude_defaults": True}])
def test_excluded_field_keeps_unknown_field_named_like_it(options):
    model = shared.FieldEnvelopeInput.model_validate({"label": "root", "plainChoice": None, "plain_choice": "unknown"})
    assert model.to_dict(**options) == {"label": "root", "plain_choice": "unknown"}
    assert model.model_dump(by_alias=True, **options) == {"label": "root", "plain_choice": "unknown"}


def test_drop_unexpected_fields_ignores_only_undeclared_fields():
    expected = shared.TypedFields(label="typed")
    assert drop_unexpected_fields(shared.TypedFields.model_validate({"label": "typed", "future": "value"}), expected) != expected
    assert drop_unexpected_fields(shared.FieldNode.model_validate({"displayName": "child", "future": True}), shared.FieldNode(display_name="child")) == shared.FieldNode(display_name="child")


def test_drop_unexpected_fields_in_typed_additional_properties():
    expected = shared.TypedNodes.model_validate({"label": "nodes", "one": {"displayName": "child"}})
    nested_unknown = shared.TypedNodes.model_validate({"label": "nodes", "one": {"displayName": "child", "future": True}})
    extra_entry = shared.TypedNodes.model_validate({"label": "nodes", "one": {"displayName": "child"}, "two": {"displayName": "other"}})
    changed_value = shared.TypedNodes.model_validate({"label": "nodes", "one": {"displayName": "changed"}})
    assert drop_unexpected_fields(nested_unknown, expected) == expected
    assert drop_unexpected_fields(extra_entry, expected) != expected
    assert drop_unexpected_fields(changed_value, expected) != expected


def test_model_serialization_methods_field_names():
    model = shared.FieldEnvelopeInput.model_validate({"label": "root", "child": {"displayName": "child", "child": {"displayName": "grandchild"}}, "futureKind": "third", "plainChoice": {"text": "t"}, "future": True})
    api_names = {"label": "root", "child": {"displayName": "child", "child": {"displayName": "grandchild"}}, "futureKind": "third", "plainChoice": {"text": "t"}, "future": True}
    field_names = {"label": "root", "child": {"display_name": "child", "child": {"display_name": "grandchild"}}, "future_kind": "third", "plain_choice": {"text": "t"}, "future": True}
    assert model.model_dump(exclude_unset=True) == api_names
    assert model.to_dict() == api_names
    assert model.to_dict(use_api_names=False) == field_names
    assert json.loads(model.to_json(use_api_names=False)) == field_names
    assert shared.FieldEnvelopeInput.model_validate(model.to_dict(use_api_names=False)).to_dict() == api_names


@pytest.mark.parametrize("typ, value", [(UnrecognizedStr, "future"), (UnrecognizedInt, 3)])
def test_unrecognized_enum_serialization(typ, value):
    adapter = pydantic.TypeAdapter(typ)
    enum_value = adapter.validate_python(value)
    assert isinstance(enum_value, typ)
    with warnings.catch_warnings():
        warnings.simplefilter("error")
        assert adapter.dump_python(enum_value) is enum_value
        assert adapter.dump_python(enum_value, mode="json") == value
        assert json.loads(adapter.dump_json(enum_value)) == value
    with pytest.raises(pydantic.ValidationError):
        adapter.validate_python(value, strict=True)


def test_unrecognized_enum_generated_model():
    model = shared.FieldEnvelope.model_validate({"label": "root", "futureKind": "third", "futureCode": 3})
    with warnings.catch_warnings():
        warnings.simplefilter("error")
        dumped = model.model_dump(by_alias=True)
        assert type(dumped["futureKind"]) is UnrecognizedStr
        assert type(dumped["futureCode"]) is UnrecognizedInt
        assert json.loads(utils.marshal_json(model, shared.FieldEnvelope)) == {"label": "root", "futureKind": "third", "futureCode": 3}


def test_flattened_parameter_named_like_helper():
    with SDK(server_url=HTTPBIN_URL) as sdk:
        response = sdk.to_json.echo_grouped_fields(to_json="j")
    assert response.raw_response.request.url.params["toJson"] == "j"
