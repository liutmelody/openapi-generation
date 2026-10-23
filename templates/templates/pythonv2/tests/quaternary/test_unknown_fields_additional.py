import json
import warnings

import pydantic
import pytest

from openapi import SDK, utils
from openapi.models import shared
from openapi.types import UnrecognizedInt, UnrecognizedStr
from .common_helpers import HTTPBIN_URL, record_test


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


def test_flattened_request_keeps_unknown_fields_on_nested_models():
    child = shared.FieldNode(display_name="child", future=None)
    with SDK(server_url=HTTPBIN_URL) as sdk:
        response = sdk.echo_flattened_unknown_fields(label="root", child=child)
    assert response.raw_response.json()["json"] == {"label": "root", "child": {"displayName": "child", "future": None}}


def test_flattened_request_has_no_argument_for_top_level_unknown_fields():
    with SDK(server_url=HTTPBIN_URL) as sdk:
        with pytest.raises(TypeError):
            sdk.echo_flattened_unknown_fields(label="root", future=True)  # type: ignore[call-arg]


@pytest.mark.parametrize("options", [{}, {"use_api_names": False}, {"exclude_unset": False}, {"exclude_defaults": True}, {"exclude_none": True}, {"warnings": False}])
def test_model_serialization_methods(options):
    model = shared.FieldEnvelopeInput.model_validate({"label": "root", "child": {"displayName": "child", "future": None}, "futureKind": "third", "futureCode": 3})
    dump_options = {"by_alias": True, "exclude_unset": True, **options}
    if "use_api_names" in dump_options:
        dump_options["by_alias"] = dump_options.pop("use_api_names")
    assert model.to_dict(**options) == model.model_dump(**dump_options)
    assert model.to_dict(mode="json", **options) == model.model_dump(mode="json", **dump_options)
    assert model.to_json(indent=None, **options) == model.model_dump_json(indent=None, **dump_options)
    assert json.loads(model.to_json(**options)) == model.to_dict(mode="json", **options)


def test_model_serialization_methods_field_names():
    model = shared.FieldEnvelopeInput.model_validate({"label": "root", "child": {"displayName": "child", "child": {"displayName": "grandchild"}}, "futureKind": "third", "plainChoice": {"text": "t"}, "future": True})
    api_names = {"label": "root", "child": {"displayName": "child", "child": {"displayName": "grandchild"}}, "futureKind": "third", "plainChoice": {"text": "t"}, "future": True}
    field_names = {"label": "root", "child": {"display_name": "child", "child": {"display_name": "grandchild"}}, "future_kind": "third", "plain_choice": {"text": "t"}, "future": True}
    assert model.model_dump(exclude_unset=True) == api_names
    assert model.to_dict() == api_names
    assert model.to_dict(use_api_names=False) == field_names
    assert json.loads(model.to_json(use_api_names=False)) == field_names
    assert shared.FieldEnvelopeInput.model_validate(model.to_dict(use_api_names=False)).to_dict() == api_names


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
