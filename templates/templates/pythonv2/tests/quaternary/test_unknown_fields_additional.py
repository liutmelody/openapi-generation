import pytest

from openapi import SDK, AsyncSDK
from openapi.models import shared
from .common_helpers import HTTPBIN_URL, record_test
from .unknown_fields import drop_unexpected_fields


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
    assert response.object.json_.model_dump(mode="json", by_alias=True) == payload


@pytest.mark.asyncio()
async def test_flattened_request_keeps_nested_unknown_fields_async():
    child = {"displayName": "child", "child": {"displayName": "grandchild", "future": {"nested": [None, 0]}}}
    async with AsyncSDK(server_url=HTTPBIN_URL) as sdk:
        response = await sdk.echo_unknown_fields(label="root", child=child)
    payload = {"label": "root", "child": child}
    assert response.raw_response.json()["json"] == payload
    assert response.object.json_.model_dump(mode="json", by_alias=True) == payload


def test_flattened_request_keeps_unknown_fields_on_nested_models():
    child = shared.FieldNode(display_name="child", future=None)
    with SDK(server_url=HTTPBIN_URL) as sdk:
        response = sdk.echo_flattened_unknown_fields(label="root", child=child)
    assert response.raw_response.json()["json"] == {"label": "root", "child": {"displayName": "child", "future": None}}


def test_flattened_request_has_no_argument_for_top_level_unknown_fields():
    with SDK(server_url=HTTPBIN_URL) as sdk:
        with pytest.raises(TypeError):
            sdk.echo_flattened_unknown_fields(label="root", future=True)  # type: ignore[call-arg]


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
