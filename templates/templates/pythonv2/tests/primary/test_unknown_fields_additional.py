import json

import pytest

from openapi import SDK, utils
from openapi.models import shared
from .common_helpers import HTTPBIN_URL
from .unknown_fields_cases import UNKNOWN_FIELD_CASES


@pytest.mark.parametrize("fields, known_fields", UNKNOWN_FIELD_CASES)
def test_unknown_fields_dropped_by_default(fields, known_fields):
    model = shared.FieldEnvelopeInput.model_validate({"label": "root", **fields})
    expected = {"label": "root", **known_fields}
    assert model.model_dump(mode="json", by_alias=True, exclude_none=True) == expected
    assert json.loads(utils.marshal_json(model, shared.FieldEnvelopeInput)) == expected


def test_flattened_request_drops_nested_unknown_fields_by_default():
    with SDK(server_url=HTTPBIN_URL) as sdk:
        response = sdk.echo_flattened_unknown_fields(label="root", child={"displayName": "child", "future": True})
    assert response.http_meta.response.json()["json"] == {"label": "root", "child": {"displayName": "child"}}
    assert response.object.model_dump(by_alias=True, exclude_none=True) == {"json": {"label": "root", "child": {"displayName": "child"}}}
