import json
from typing import Annotated, Any, List, Optional

import httpx
import pydantic
import pytest

from openapi import SDK, utils
from openapi.models import errors, shared
from openapi.types import UNSET, BaseModel, Nullable
from .common_helpers import HTTPBIN_URL, record_test


CASES = [
    {"futureKind": "third", "futureCode": 3},
    {"methodNames": {"toDict": "dictionary", "toJson": "json"}},
    {"future": "value"},
    {"future": False},
    {"empty": {"future": True}},
    {"future": None},
    {"plainChoice": {"text": "value", "future": True}},
    {"plainChoice": {"number": 3, "future": True}},
    {"requiredOnly": {"label": "required", "future": True}},
    {"future": 0},
    {"composed": {"displayName": "child", "category": "combined", "future": True}},
    {"split": {"label": "split", "future": True}},
    {"future": []},
    {"future": {"nested": [None, False, 0, "value"]}},
    {"child": {"displayName": "child", "future": True}},
    {"child": {"displayName": "child", "display_name": "unknown"}},
    {"child": {"displayName": "child", "child": {"displayName": "grandchild", "future": True}}},
    {"children": [{"displayName": "child", "future": True}]},
    {"lookup": {"one": {"displayName": "child", "future": True}}},
    {"choice": {"kind": "text", "text": "value", "future": True}},
    {"choice": {"kind": "number", "number": 3, "future": True}},
    {"closed": {"label": "closed", "future": True}},
    {"open": {"label": "open", "future": True}},
    {"typed": {"label": "typed", "future": "value"}},
    {"arbitrary": {"future": {"nullable": None}}},
    {"child": {"displayName": "child", "nullable": None}},
]


def echo(request):
    sent = []
    with httpx.Client(event_hooks={"request": [sent.append]}) as client:
        with SDK(server_url=HTTPBIN_URL, client=client) as sdk:
            response = sdk.echo_unknown_fields(request=request)
    return response, json.loads(sent[-1].content)


@pytest.mark.parametrize("fields", CASES)
@pytest.mark.parametrize("as_model", [False, True])
def test_unknown_fields_echo(fields, as_model):
    record_test("unknown-fields-json-round-trip")
    payload = {"label": "root", **fields}
    request = shared.FieldEnvelope.model_validate(payload) if as_model else payload
    response, sent = echo(request)
    assert sent == payload
    assert response.json_.model_dump(mode="json", by_alias=True) == payload
    assert json.loads(response.json_.model_dump_json(by_alias=True)) == payload
    assert "headers" in response.model_dump()


def test_response_only_unknown_fields():
    payload = {"json": {"label": "root", "child": {"displayName": "child", "future": True}}, "future": {"value": 1}}
    transport = httpx.MockTransport(lambda request: httpx.Response(200, json=payload))
    with httpx.Client(transport=transport) as client:
        with SDK(server_url=HTTPBIN_URL, client=client) as sdk:
            response = sdk.echo_unknown_fields(request={"label": "root"})
    dumped = response.model_dump(mode="json", by_alias=True)
    assert "future" in dumped
    assert "future" in dumped["json"]["child"]


def test_response_model_can_be_sent_again():
    payload = {"label": "root", "child": {"displayName": "child", "future": {"enabled": False}}}
    first, _ = echo(payload)
    second, sent = echo(first.json_.model_dump(by_alias=True))
    assert sent == payload
    assert second.json_.model_dump(by_alias=True, exclude_none=True) == payload


def test_error_response_unknown_fields():
    payload = {"message": "invalid", "future": None, "detail": {"displayName": "child", "future": {"flag": True}}}
    transport = httpx.MockTransport(lambda request: httpx.Response(400, json=payload))
    with httpx.Client(transport=transport) as client:
        with SDK(server_url=HTTPBIN_URL, client=client) as sdk:
            with pytest.raises(errors.EchoUnknownFieldsResponseBody) as caught:
                sdk.echo_unknown_fields(request={"label": "root"})
    dumped = getattr(caught.value, "data").model_dump(mode="json", by_alias=True)
    assert "future" in dumped
    assert "future" in dumped["detail"]


def test_omitted_json_query_stays_empty():
    sent = []
    with httpx.Client(event_hooks={"request": [sent.append]}) as client:
        with SDK(server_url=HTTPBIN_URL, client=client) as sdk:
            sdk.echo_optional_json_query()
    assert sent[-1].url.params["filter"] == ""


def test_typed_additional_properties_still_validate():
    with pytest.raises(pydantic.ValidationError):
        shared.FieldEnvelope.model_validate({"label": "root", "typed": {"future": {"invalid": True}}})


def test_unknown_fields_dump_filters():
    model = shared.FieldEnvelope.model_validate({"label": "root", "future": True, "future_null": None})
    assert "future_null" in model.model_dump()
    assert "future_null" not in model.model_dump(exclude_none=True)
    assert "future" not in model.model_dump(exclude={"future"})
    assert model.model_dump(include={"label"}) == {"label": "root"}


def test_base_model_unknown_field_policy():
    class CustomModel(BaseModel):
        label: str

    model = CustomModel(label="value", future=None)
    assert "future" in model.model_dump()


def test_unset_ignores_unknown_fields():
    node = shared.FieldNode.model_validate({"displayName": "child", "nullable": {"future": True}})
    assert node.nullable == UNSET


def test_helper_names_not_reserved_without_methods():
    assert set(shared.MethodNames.model_fields) == {"to_dict", "to_json"}
    assert not hasattr(shared.FieldEnvelope, "to_dict")


@pytest.mark.parametrize("typ", [Optional[int], Optional[List[int]], Optional[shared.FieldNode], Any])
def test_top_level_none_serialization(typ):
    assert utils.marshal_json(None, typ) == ""


def test_top_level_none_still_validates():
    with pytest.raises(pydantic.ValidationError):
        utils.marshal_json(None, shared.FieldNode)
    assert utils.marshal_json(None, Nullable[shared.FieldNode]) == "null"


def test_coerced_top_level_none_stays_empty():
    typ = Annotated[Optional[int], pydantic.BeforeValidator(lambda value: None)]
    assert utils.marshal_json("empty", typ) == ""
