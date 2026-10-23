import json

from openapi.models.shared import OptionalNestedRequest, OptionalNestedResponse


def test_optional_nested_request_preserves_additional_properties():
    payload = {
        "configuration": {
            "options": {"kind": "structured", "custom_options": True},
            "custom_configuration": "keep",
        },
        "custom_request": 7,
    }
    request = OptionalNestedRequest.model_validate(payload)

    assert json.loads(request.model_dump_json(by_alias=True)) == payload
    assert "configuration" not in OptionalNestedRequest().model_dump(by_alias=True)
    assert OptionalNestedRequest(configuration=None).model_dump(by_alias=True)["configuration"] is None


def test_optional_nested_response_preserves_usage():
    payload = {
        "usage": {
            "total": 7,
            "details": {"reused": 0, "custom_details": 2},
            "custom_usage": "keep",
        },
        "custom_response": True,
    }
    response = OptionalNestedResponse.model_validate(payload)

    assert response.usage.total == 7
    assert response.model_dump(by_alias=True) == payload
    assert json.loads(response.model_dump_json(by_alias=True)) == payload
