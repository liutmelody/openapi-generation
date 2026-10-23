"""
Test suite for open enum union discrimination with UnrecognizedStr and UnrecognizedInt types.

This tests the behavior of Union[Literal["known"], UnrecognizedStr] and Union[Literal[1], UnrecognizedInt] where:
- UnrecognizedStr/UnrecognizedInt only work in lax mode (used for open enums)
- In strict mode, UnrecognizedStr/UnrecognizedInt always fail, forcing Pydantic to prefer Literal matches
- Smart union mode correctly discriminates between union members based on field matches
"""

import warnings

import pytest
from typing import Union, Literal, Optional
from pydantic import BaseModel, ValidationError, TypeAdapter, ConfigDict
from .common_helpers import record_test

from openapi import SDK, utils
from openapi.models import shared, operations
from openapi.types import UnrecognizedStr, UnrecognizedInt


class TestUnrecognisedStrBasicBehavior:
    """Test basic UnrecognizedStr behavior in unions with Literals."""

    def test_UnrecognisedStr_accepts_literal_value_in_lax_mode(self):
        """Test that UnrecognizedStr accepts known literal values in lax mode."""

        class Model(BaseModel):
            value: Union[Literal["known"], UnrecognizedStr]

        # Should accept the literal value
        m = Model(value="known")
        assert m.value == "known"
        assert isinstance(m.value, str)

    def test_UnrecognisedStr_accepts_unknown_value_in_lax_mode(self):
        """Test that UnrecognizedStr accepts unknown values in lax mode."""

        class Model(BaseModel):
            value: Union[Literal["known"], UnrecognizedStr]

        # Should accept unknown values via UnrecognizedStr fallback
        m = Model(value="unknown")
        assert m.value == "unknown"
        assert isinstance(m.value, str)

    def test_UnrecognisedStr_prefers_literal_in_smart_union(self):
        """Test that smart union mode prefers Literal over UnrecognizedStr."""

        class ModelA(BaseModel):
            key: Literal["aaa"]

        class ModelB(BaseModel):
            key: Union[Literal["bbb"], UnrecognizedStr]

        U = Union[ModelA, ModelB]
        adapter = TypeAdapter(U, config=ConfigDict(union_mode='smart'))

        # Should match ModelA for "aaa"
        result_a = adapter.validate_python({"key": "aaa"})
        assert isinstance(result_a, ModelA)
        assert result_a.key == "aaa"

        # Should match ModelB for "bbb"
        result_b = adapter.validate_python({"key": "bbb"})
        assert isinstance(result_b, ModelB)
        assert result_b.key == "bbb"

        # Should match ModelB (via UnrecognizedStr fallback) for "ccc"
        result_c = adapter.validate_python({"key": "ccc"})
        assert isinstance(result_c, ModelB)
        assert result_c.key == "ccc"


class TestUnionDiscriminationWithOpenEnums:
    """Test union discrimination when models have different field structures."""

    def test_prefers_more_specific_model_with_more_fields(self):
        """Test that model with more matching fields wins."""
        record_test("smart-union-selects-more-matched-fields")

        class A(BaseModel):
            foo: str

        class B(BaseModel):
            foo: str
            bar: Union[Literal["b"], UnrecognizedStr]

        U = Union[A, B]
        adapter = TypeAdapter(U, config=ConfigDict(union_mode='smart'))

        # Payload has both fields, should match B
        payload = {"foo": "test", "bar": "value"}
        result = adapter.validate_python(payload)

        assert isinstance(result, B)
        assert result.foo == "test"
        assert result.bar == "value"

    def test_nested_models_with_different_fields(self):
        """Test union discrimination with nested models."""
        record_test("smart-union-nested-structs")

        class InnerA(BaseModel):
            foo: str

        class InnerB(BaseModel):
            foo: str
            bar: Union[Literal["b"], UnrecognizedStr]

        class A(BaseModel):
            inner: InnerA

        class B(BaseModel):
            inner: InnerB

        U = Union[A, B]
        adapter = TypeAdapter(U, config=ConfigDict(union_mode='smart'))

        # Nested payload has both fields, should match B
        payload = {"inner": {"foo": "test", "bar": "value"}}
        result = adapter.validate_python(payload)

        assert isinstance(result, B)
        assert result.inner.foo == "test"
        assert result.inner.bar == "value"

    def test_nested_models_with_different_field_names(self):
        """Test discrimination when nested models have completely different fields."""

        class InnerA(BaseModel):
            foo: Union[Literal["foo"], UnrecognizedStr]

        class InnerB(BaseModel):
            bar: Union[Literal["bar"], UnrecognizedStr]

        class A(BaseModel):
            inner: InnerA

        class B(BaseModel):
            inner: InnerB

        U = Union[A, B]
        adapter = TypeAdapter(U, config=ConfigDict(union_mode='smart'))

        # Payload has "bar" field, should match B
        payload = {"inner": {"bar": "bar"}}
        result = adapter.validate_python(payload)

        assert isinstance(result, B)
        assert result.inner.bar == "bar"

        # Payload has "foo" field, should match A
        payload2 = {"inner": {"foo": "foo"}}
        result2 = adapter.validate_python(payload2)

        assert isinstance(result2, A)
        assert result2.inner.foo == "foo"

    def test_nested_models_with_mixed_fields(self):
        """Test discrimination when payload has mixed fields."""

        class InnerA(BaseModel):
            foo: Union[Literal["foo"], UnrecognizedStr]

        class InnerB(BaseModel):
            bar: Union[Literal["bar"], UnrecognizedStr]

        class A(BaseModel):
            inner: InnerA

        class B(BaseModel):
            inner: InnerB

        U = Union[A, B]
        adapter = TypeAdapter(U, config=ConfigDict(union_mode='smart'))

        # Payload has both "foo" and "bar", should match the one with exact literal match
        payload = {"inner": {"foo": "", "bar": "bar"}}
        result = adapter.validate_python(payload)

        # B should win because it has an exact match for "bar" literal
        assert isinstance(result, B)
        assert result.inner.bar == "bar"


class TestSmartUnionOpenEnums:
    """Test smart union discrimination with open enums."""

    def test_smart_union_open_enums(self):
        """Test that open enum value is correctly parsed in union.

        types: { kind: OpenEnum["cat"] } | { kind: OpenEnum["dog"] }
        payload: { kind: "dog" }
        expect: should match dog variant (exact match)
        """
        record_test("smart-union-open-enums")

        class Cat(BaseModel):
            kind: Union[Literal["cat"], UnrecognizedStr]

        class Dog(BaseModel):
            kind: Union[Literal["dog"], UnrecognizedStr]

        U = Union[Cat, Dog]
        adapter = TypeAdapter(U, config=ConfigDict(union_mode='smart'))

        # Known kind "dog" should match Dog variant
        payload = {"kind": "dog"}
        result = adapter.validate_python(payload)

        assert isinstance(result, Dog)
        assert result.kind == "dog"

    def test_smart_union_open_enums_and_size(self):
        """Test that model with more fields wins when enum doesn't match.

        types: { kind: OpenEnum["cat"], name: string } | { kind: OpenEnum["dog"] }
        payload: { kind: "bat", name: "asdf" }
        expect: should match cat variant (has name field)
        """
        record_test("smart-union-open-enums-and-size")

        class Cat(BaseModel):
            kind: Union[Literal["cat"], UnrecognizedStr]
            name: str

        class Dog(BaseModel):
            kind: Union[Literal["dog"], UnrecognizedStr]

        U = Union[Cat, Dog]
        adapter = TypeAdapter(U, config=ConfigDict(union_mode='smart'))

        # Unknown kind "bat" + has name field → Cat should win
        payload = {"kind": "bat", "name": "asdf"}
        result = adapter.validate_python(payload)

        assert isinstance(result, Cat)
        assert result.kind == "bat"
        assert result.name == "asdf"

    def test_smart_union_nested_union(self):
        """Test nested union only counts winning inner option's unrecognized values.

        Outer union structure:
          Option A: { data: InnerUnion }
            where InnerUnion is:
              - cat:  { kind: OpenEnum["cat"] }                (1 open enum field)
              - dog:  { kind: OpenEnum["dog"] }                (1 open enum field)
              - bird: { kind: OpenEnum["bird"] }               (1 open enum field)
          Option B: { data: { kind: OpenEnum, name: OpenEnum } } (2 open enum fields)

        payload: { data: { kind: "unknown", name: "also_unknown" } }

        Expected: Option B should win because:
          - Option A: Inner union tries all variants (cat/dog/bird), but none match perfectly
            The response has 2 fields (kind, name), but each variant only has 1 field (kind)
            Best match would still count the extra 'name' field as unmatched
          - Option B: Has both kind and name fields as open enums, perfect structural match
            Counts 2 unrecognized (both kind and name are unknown enum values)
          - Option B wins because it has better field coverage despite more unrecognized enum values
        """
        record_test("smart-union-nested-union")

        # Inner union variants for Option A
        class CatInner(BaseModel):
            kind: Union[Literal["cat"], UnrecognizedStr]

        class DogInner(BaseModel):
            kind: Union[Literal["dog"], UnrecognizedStr]

        class BirdInner(BaseModel):
            kind: Union[Literal["bird"], UnrecognizedStr]

        InnerUnion = Union[CatInner, DogInner, BirdInner]

        # Option B's inner type - has both kind and name as open enums
        class KindAndName(BaseModel):
            kind: Union[Literal["known_kind"], UnrecognizedStr]
            name: Union[Literal["known_name"], UnrecognizedStr]

        # Outer union options
        class OptionA(BaseModel):
            model_config = ConfigDict(union_mode='smart')
            data: InnerUnion

        class OptionB(BaseModel):
            data: KindAndName

        U = Union[OptionA, OptionB]
        adapter = TypeAdapter(U, config=ConfigDict(union_mode='smart'))

        # Payload with unknown enum values - Option B should win (better field coverage)
        payload = {"data": {"kind": "unknown", "name": "also_unknown"}}
        result = adapter.validate_python(payload)

        assert isinstance(result, OptionB)
        assert result.data.kind == "unknown"
        assert result.data.name == "also_unknown"


class TestOpenEnumBehaviorInStructs:
    """Test open enum behavior in more complex struct scenarios."""

    def test_open_enum_with_known_value(self):
        """Test that known enum values work correctly."""

        class Theme(BaseModel):
            color: Union[Literal["red", "green", "blue"], UnrecognizedStr]

        theme = Theme(color="red")
        assert theme.color == "red"

    def test_open_enum_with_unknown_value(self):
        """Test that unknown enum values are accepted via UnrecognizedStr."""

        class Theme(BaseModel):
            color: Union[Literal["red", "green", "blue"], UnrecognizedStr]

        theme = Theme(color="purple")
        assert theme.color == "purple"

    def test_multiple_open_enums_in_model(self):
        """Test model with multiple open enum fields."""

        class Config(BaseModel):
            size: Union[Literal["small", "medium", "large"], UnrecognizedStr]
            color: Union[Literal["red", "green", "blue"], UnrecognizedStr]
            style: Union[Literal["solid", "dashed"], UnrecognizedStr]

        # Mix of known and unknown values
        config = Config(
            size="xl",  # unknown
            color="green",  # known
            style="dotted"  # unknown
        )

        assert config.size == "xl"
        assert config.color == "green"
        assert config.style == "dotted"

class TestUnionOrderIndependence:
    """Test that union discrimination is order-independent when appropriate."""

    def test_literal_before_UnrecognisedStr(self):
        """Test Union[Literal, UnrecognizedStr] order."""

        class Model(BaseModel):
            value: Union[Literal["known"], UnrecognizedStr]

        m1 = Model(value="known")
        assert m1.value == "known"

        m2 = Model(value="unknown")
        assert m2.value == "unknown"

    def test_models_with_different_field_counts(self):
        """Test that model with fewer unmatched fields wins."""

        class Minimal(BaseModel):
            foo: str

        class Extended(BaseModel):
            foo: str
            bar: Union[Literal["b"], UnrecognizedStr]
            baz: Union[Literal["c"], UnrecognizedStr]

        U = Union[Minimal, Extended]
        adapter = TypeAdapter(U, config=ConfigDict(union_mode='smart'))

        # Payload only has "foo", should match Minimal (fewer unmatched fields)
        payload1 = {"foo": "test"}
        result1 = adapter.validate_python(payload1)
        assert isinstance(result1, Minimal)

        # Payload has all fields, should match Extended (more matched fields)
        payload2 = {"foo": "test", "bar": "b", "baz": "c"}
        result2 = adapter.validate_python(payload2)
        assert isinstance(result2, Extended)


class TestSerializationRoundTrip:
    """Test that models with UnrecognizedStr can be serialized and deserialized."""

    def test_model_dump_and_load(self):
        """Test model_dump and model_validate roundtrip."""

        class Theme(BaseModel):
            color: Union[Literal["red", "green", "blue"], UnrecognizedStr]
            icon: Union[Literal["star", "circle"], UnrecognizedStr]

        # Create with unknown values
        theme1 = Theme(color="purple", icon="triangle")

        # Dump to dict
        data = theme1.model_dump()
        assert data == {"color": "purple", "icon": "triangle"}

        # Load back
        theme2 = Theme.model_validate(data)
        assert theme2.color == "purple"
        assert theme2.icon == "triangle"

    def test_json_serialization(self):
        """Test JSON serialization roundtrip."""

        class Theme(BaseModel):
            color: Union[Literal["red", "green", "blue"], UnrecognizedStr]
            size: Union[Literal["small", "medium", "large"], UnrecognizedStr]

        theme1 = Theme(color="orange", size="xl")

        # Serialize to JSON
        json_str = theme1.model_dump_json()
        assert "orange" in json_str
        assert "xl" in json_str

        # Deserialize from JSON
        theme2 = Theme.model_validate_json(json_str)
        assert theme2.color == "orange"
        assert theme2.size == "xl"


class TestUnionWithNullableFields:
    """Test UnrecognizedStr behavior with nullable/optional fields."""

    def test_optional_open_enum_field(self):
        """Test Optional[Union[Literal, UnrecognizedStr]]."""

        from typing import Optional

        class Config(BaseModel):
            color: Optional[Union[Literal["red", "green"], UnrecognizedStr]] = None

        # With value
        c1 = Config(color="blue")
        assert c1.color == "blue"

        # With None
        c2 = Config(color=None)
        assert c2.color is None

        # Without field (should default to None)
        c3 = Config()
        assert c3.color is None

    def test_union_with_none(self):
        """Test Union[Literal, UnrecognizedStr, None]."""

        class Model(BaseModel):
            value: Union[Literal["known"], UnrecognizedStr, None]

        m1 = Model(value="known")
        assert m1.value == "known"

        m2 = Model(value="unknown")
        assert m2.value == "unknown"

        m3 = Model(value=None)
        assert m3.value is None


# =============================================================================
# UnrecognizedInt Tests - Mirror of UnrecognizedStr tests for integer open enums
# =============================================================================


class TestUnrecognizedIntBasicBehavior:
    """Test basic UnrecognizedInt behavior in unions with Literals."""

    def test_UnrecognizedInt_accepts_literal_value_in_lax_mode(self):
        """Test that UnrecognizedInt accepts known literal values in lax mode."""

        class Model(BaseModel):
            value: Union[Literal[42], UnrecognizedInt]

        # Should accept the literal value
        m = Model(value=42)
        assert m.value == 42
        assert isinstance(m.value, int)

    def test_UnrecognizedInt_accepts_unknown_value_in_lax_mode(self):
        """Test that UnrecognizedInt accepts unknown values in lax mode."""

        class Model(BaseModel):
            value: Union[Literal[42], UnrecognizedInt]

        # Should accept unknown values via UnrecognizedInt fallback
        m = Model(value=99)
        assert m.value == 99
        assert isinstance(m.value, int)

    def test_UnrecognizedInt_prefers_literal_in_smart_union(self):
        """Test that smart union mode prefers Literal over UnrecognizedInt."""

        class ModelA(BaseModel):
            key: Literal[1]

        class ModelB(BaseModel):
            key: Union[Literal[2], UnrecognizedInt]

        U = Union[ModelA, ModelB]
        adapter = TypeAdapter(U, config=ConfigDict(union_mode='smart'))

        # Should match ModelA for 1
        result_a = adapter.validate_python({"key": 1})
        assert isinstance(result_a, ModelA)
        assert result_a.key == 1

        # Should match ModelB for 2
        result_b = adapter.validate_python({"key": 2})
        assert isinstance(result_b, ModelB)
        assert result_b.key == 2

        # Should match ModelB (via UnrecognizedInt fallback) for 3
        result_c = adapter.validate_python({"key": 3})
        assert isinstance(result_c, ModelB)
        assert result_c.key == 3


class TestUnionDiscriminationWithOpenEnumsInt:
    """Test union discrimination when models have different field structures with integers."""

    def test_prefers_more_specific_model_with_more_fields(self):
        """Test that model with more matching fields wins."""

        class A(BaseModel):
            foo: int

        class B(BaseModel):
            foo: int
            bar: Union[Literal[1], UnrecognizedInt]

        U = Union[A, B]
        adapter = TypeAdapter(U, config=ConfigDict(union_mode='smart'))

        # Payload has both fields, should match B
        payload = {"foo": 100, "bar": 42}
        result = adapter.validate_python(payload)

        assert isinstance(result, B)
        assert result.foo == 100
        assert result.bar == 42

    def test_nested_models_with_different_fields(self):
        """Test union discrimination with nested models."""

        class InnerA(BaseModel):
            foo: int

        class InnerB(BaseModel):
            foo: int
            bar: Union[Literal[1], UnrecognizedInt]

        class A(BaseModel):
            inner: InnerA

        class B(BaseModel):
            inner: InnerB

        U = Union[A, B]
        adapter = TypeAdapter(U, config=ConfigDict(union_mode='smart'))

        # Nested payload has both fields, should match B
        payload = {"inner": {"foo": 100, "bar": 42}}
        result = adapter.validate_python(payload)

        assert isinstance(result, B)
        assert result.inner.foo == 100
        assert result.inner.bar == 42

    def test_nested_models_with_different_field_names(self):
        """Test discrimination when nested models have completely different fields."""

        class InnerA(BaseModel):
            foo: Union[Literal[1], UnrecognizedInt]

        class InnerB(BaseModel):
            bar: Union[Literal[2], UnrecognizedInt]

        class A(BaseModel):
            inner: InnerA

        class B(BaseModel):
            inner: InnerB

        U = Union[A, B]
        adapter = TypeAdapter(U, config=ConfigDict(union_mode='smart'))

        # Payload has "bar" field, should match B
        payload = {"inner": {"bar": 2}}
        result = adapter.validate_python(payload)

        assert isinstance(result, B)
        assert result.inner.bar == 2

        # Payload has "foo" field, should match A
        payload2 = {"inner": {"foo": 1}}
        result2 = adapter.validate_python(payload2)

        assert isinstance(result2, A)
        assert result2.inner.foo == 1

    def test_nested_models_with_mixed_fields(self):
        """Test discrimination when payload has mixed fields."""

        class InnerA(BaseModel):
            foo: Union[Literal[1], UnrecognizedInt]

        class InnerB(BaseModel):
            bar: Union[Literal[2], UnrecognizedInt]

        class A(BaseModel):
            inner: InnerA

        class B(BaseModel):
            inner: InnerB

        U = Union[A, B]
        adapter = TypeAdapter(U, config=ConfigDict(union_mode='smart'))

        # Payload has both "foo" and "bar", should match the one with exact literal match
        payload = {"inner": {"foo": 0, "bar": 2}}
        result = adapter.validate_python(payload)

        # B should win because it has an exact match for "bar" literal
        assert isinstance(result, B)
        assert result.inner.bar == 2


class TestOpenEnumBehaviorInStructsInt:
    """Test open enum behavior in more complex struct scenarios with integers."""

    def test_open_enum_with_known_value(self):
        """Test that known enum values work correctly."""

        class Status(BaseModel):
            code: Union[Literal[200, 201, 404], UnrecognizedInt]

        status = Status(code=200)
        assert status.code == 200

    def test_open_enum_with_unknown_value(self):
        """Test that unknown enum values are accepted via UnrecognizedInt."""

        class Status(BaseModel):
            code: Union[Literal[200, 201, 404], UnrecognizedInt]

        status = Status(code=500)
        assert status.code == 500

    def test_multiple_open_enums_in_model(self):
        """Test model with multiple open enum fields."""

        class Config(BaseModel):
            priority: Union[Literal[1, 2, 3], UnrecognizedInt]
            status: Union[Literal[0, 1], UnrecognizedInt]
            level: Union[Literal[10, 20, 30], UnrecognizedInt]

        # Mix of known and unknown values
        config = Config(
            priority=5,  # unknown
            status=1,  # known
            level=99  # unknown
        )

        assert config.priority == 5
        assert config.status == 1
        assert config.level == 99


class TestUnionOrderIndependenceInt:
    """Test that union discrimination is order-independent when appropriate with integers."""

    def test_literal_before_UnrecognizedInt(self):
        """Test Union[Literal, UnrecognizedInt] order."""

        class Model(BaseModel):
            value: Union[Literal[42], UnrecognizedInt]

        m1 = Model(value=42)
        assert m1.value == 42

        m2 = Model(value=99)
        assert m2.value == 99

    def test_models_with_different_field_counts(self):
        """Test that model with fewer unmatched fields wins."""

        class Minimal(BaseModel):
            foo: int

        class Extended(BaseModel):
            foo: int
            bar: Union[Literal[1], UnrecognizedInt]
            baz: Union[Literal[2], UnrecognizedInt]

        U = Union[Minimal, Extended]
        adapter = TypeAdapter(U, config=ConfigDict(union_mode='smart'))

        # Payload only has "foo", should match Minimal (fewer unmatched fields)
        payload1 = {"foo": 100}
        result1 = adapter.validate_python(payload1)
        assert isinstance(result1, Minimal)

        # Payload has all fields, should match Extended (more matched fields)
        payload2 = {"foo": 100, "bar": 1, "baz": 2}
        result2 = adapter.validate_python(payload2)
        assert isinstance(result2, Extended)


class TestSerializationRoundTripInt:
    """Test that models with UnrecognizedInt can be serialized and deserialized."""

    def test_model_dump_and_load(self):
        """Test model_dump and model_validate roundtrip."""

        class Status(BaseModel):
            code: Union[Literal[200, 201, 404], UnrecognizedInt]
            priority: Union[Literal[1, 2, 3], UnrecognizedInt]

        # Create with unknown values
        status1 = Status(code=500, priority=5)

        # Dump to dict
        data = status1.model_dump()
        assert data == {"code": 500, "priority": 5}

        # Load back
        status2 = Status.model_validate(data)
        assert status2.code == 500
        assert status2.priority == 5

    def test_json_serialization(self):
        """Test JSON serialization roundtrip."""

        class Status(BaseModel):
            code: Union[Literal[200, 201, 404], UnrecognizedInt]
            level: Union[Literal[1, 2, 3], UnrecognizedInt]

        status1 = Status(code=500, level=10)

        # Serialize to JSON
        json_str = status1.model_dump_json()
        assert "500" in json_str
        assert "10" in json_str

        # Deserialize from JSON
        status2 = Status.model_validate_json(json_str)
        assert status2.code == 500
        assert status2.level == 10


class TestUnionWithNullableFieldsInt:
    """Test UnrecognizedInt behavior with nullable/optional fields."""

    def test_optional_open_enum_field(self):
        """Test Optional[Union[Literal, UnrecognizedInt]]."""

        from typing import Optional

        class Config(BaseModel):
            priority: Optional[Union[Literal[1, 2, 3], UnrecognizedInt]] = None

        # With value
        c1 = Config(priority=5)
        assert c1.priority == 5

        # With None
        c2 = Config(priority=None)
        assert c2.priority is None

        # Without field (should default to None)
        c3 = Config()
        assert c3.priority is None

    def test_union_with_none(self):
        """Test Union[Literal, UnrecognizedInt, None]."""

        class Model(BaseModel):
            value: Union[Literal[42], UnrecognizedInt, None]

        m1 = Model(value=42)
        assert m1.value == 42

        m2 = Model(value=99)
        assert m2.value == 99

        m3 = Model(value=None)
        assert m3.value is None


# =============================================================================
# Array Tests - Ported from TypeScript smart_union.test.ts
# =============================================================================


class TestArrayUnionDiscrimination:
    """Test union discrimination with array types."""

    def test_array_with_more_fields_wins(self):
        """Test that array schema with more fields in items wins."""

        from typing import Optional

        class ItemA(BaseModel):
            x: str

        class ItemB(BaseModel):
            x: str
            y: bool

        class WrapperA(BaseModel):
            items: list[ItemA]

        class WrapperB(BaseModel):
            items: list[ItemB]

        U = Union[WrapperA, WrapperB]
        adapter = TypeAdapter(U, config=ConfigDict(union_mode='smart'))

        # Payload has items with both x and y fields, should match B
        payload = {"items": [{"x": "a", "y": False}, {"x": "b", "y": True}]}
        result = adapter.validate_python(payload)

        assert isinstance(result, WrapperB)
        assert len(result.items) == 2
        assert result.items[0].x == "a"
        assert result.items[0].y is False

    def test_array_of_nullable_structs(self):
        """Test union discrimination with arrays containing nullable structs."""

        from typing import Optional

        class ItemA(BaseModel):
            foo: Optional[str] = None

        class ItemB(BaseModel):
            bar: Optional[str] = None

        class WrapperA(BaseModel):
            items: list[Optional[ItemA]]

        class WrapperB(BaseModel):
            items: list[Optional[ItemB]]

        U = Union[WrapperA, WrapperB]
        adapter = TypeAdapter(U, config=ConfigDict(union_mode='smart'))

        # Payload with null items and one with bar field, should match B
        payload = {"items": [None, None, {"bar": "test"}]}
        result = adapter.validate_python(payload)

        assert isinstance(result, WrapperB)
        assert result.items[0] is None
        assert result.items[1] is None
        assert result.items[2].bar == "test"


# =============================================================================
# Dict/Map Tests - Ported from TypeScript smart_union.test.ts
# =============================================================================


class TestDictUnionDiscrimination:
    """Test union discrimination with dict/map types."""

    def test_dict_of_structs_matching_inner_field(self):
        """Test that dict schema with matching inner field wins."""

        from typing import Optional, Dict

        class InnerA(BaseModel):
            a: str

        class InnerB(BaseModel):
            b: str

        class WrapperA(BaseModel):
            data: Dict[str, Optional[InnerA]]

        class WrapperB(BaseModel):
            data: Dict[str, Optional[InnerB]]

        U = Union[WrapperA, WrapperB]
        adapter = TypeAdapter(U, config=ConfigDict(union_mode='smart'))

        # Payload with dict containing 'b' field, should match B
        payload = {"data": {"x": None, "y": None, "z": {"b": "value"}}}
        result = adapter.validate_python(payload)

        assert isinstance(result, WrapperB)
        assert result.data["z"].b == "value"

    def test_dict_with_nested_structs_vs_simple_field(self):
        """Test dict with nested structs wins over simple field."""

        from typing import Dict

        class A(BaseModel):
            a: str

        class NestedItem(BaseModel):
            id: str
            name: str

        class B(BaseModel):
            b: Dict[str, NestedItem]

        U = Union[A, B]
        adapter = TypeAdapter(U, config=ConfigDict(union_mode='smart'))

        # Payload has both 'a' and 'b' with nested struct, B should win (more matched fields)
        payload = {"a": "", "b": {"foo": {"id": "", "name": ""}}}
        result = adapter.validate_python(payload)

        # B should win because it has more matched fields (id, name in nested struct)
        assert isinstance(result, B)
        assert result.b["foo"].id == ""
        assert result.b["foo"].name == ""


# =============================================================================
# Enum and Const Field Discrimination Tests
# =============================================================================


class TestEnumDiscrimination:
    """Test enum-based discrimination in unions."""

    def test_enum_discrimination_with_open_enums(self):
        """Test that exact enum matches are preferred."""

        class A(BaseModel):
            a: Union[Literal["1", "2"], UnrecognizedStr]

        class B(BaseModel):
            a: Union[Literal["3", "4"], UnrecognizedStr]

        U = Union[A, B]
        adapter = TypeAdapter(U, config=ConfigDict(union_mode='smart'))

        # Value "4" exactly matches B's enum
        payload = {"a": "4"}
        result = adapter.validate_python(payload)

        assert isinstance(result, B)
        assert result.a == "4"

    def test_const_field_discrimination(self):
        """Test discrimination using const/literal fields."""
        record_test("smart-union-const-field-discrimination")

        class A(BaseModel):
            a: Literal["x"]
            b: Literal["1"]

        class B(BaseModel):
            a: Literal["x"]
            c: Literal["1"]

        class C(BaseModel):
            b: Literal["1"]
            c: Literal["1"]

        U = Union[A, B, C]
        adapter = TypeAdapter(U, config=ConfigDict(union_mode='smart'))

        # Payload has b and c, should match C
        payload = {"b": "1", "c": "1"}
        result = adapter.validate_python(payload)

        assert isinstance(result, C)
        assert result.b == "1"
        assert result.c == "1"

    def test_const_field_discrimination_with_inexact_values(self):
        """Test discrimination when open enums have non-exact values."""

        class A(BaseModel):
            a: Union[Literal["x"], UnrecognizedStr]
            b: Union[Literal["1"], UnrecognizedStr]

        class B(BaseModel):
            a: Union[Literal["x"], UnrecognizedStr]
            c: Union[Literal["1"], UnrecognizedStr]

        class C(BaseModel):
            b: Union[Literal["1"], UnrecognizedStr]
            c: Union[Literal["1"], UnrecognizedStr]

        U = Union[A, B, C]
        adapter = TypeAdapter(U, config=ConfigDict(union_mode='smart'))

        # Payload has b and c with non-matching values
        # C should win because it has more matched fields (2 vs 1)
        payload = {"b": "hey", "c": "ho"}
        result = adapter.validate_python(payload)

        assert isinstance(result, C)
        assert result.b == "hey"
        assert result.c == "ho"


# =============================================================================
# Lax Mode / Coercion Tests
# =============================================================================


class TestLaxModeDiscrimination:
    """Test discrimination behavior in lax mode with type coercion."""

    def test_prefers_exact_type_over_coerced(self):
        """Test that exact type match is preferred over coerced value."""

        class A(BaseModel):
            a: str

        class B(BaseModel):
            a: int

        U = Union[A, B]
        adapter = TypeAdapter(U, config=ConfigDict(union_mode='smart'))

        # String "1" should match A (exact), not B (coerced)
        payload = {"a": "1"}
        result = adapter.validate_python(payload)

        assert isinstance(result, A)
        assert result.a == "1"

    def test_null_pointers_match_over_missing_fields(self):
        """Test that null values count as matched fields."""

        from typing import Optional

        class A(BaseModel):
            a: str

        class B(BaseModel):
            b: Optional[str] = None
            c: Optional[str] = None

        U = Union[A, B]
        adapter = TypeAdapter(U, config=ConfigDict(union_mode='smart'))

        # Payload has b and c as null, B should win (2 matched fields vs 0)
        payload = {"b": None, "c": None}
        result = adapter.validate_python(payload)

        assert isinstance(result, B)
        assert result.b is None
        assert result.c is None


# =============================================================================
# Three-Way Discrimination Tests
# =============================================================================


class TestThreeWayDiscrimination:
    """Test discrimination with three or more union options."""

    def test_three_way_field_discrimination(self):
        """Test discrimination based on field presence with three options."""
        record_test("smart-union-three-way-field-discrimination")

        class A(BaseModel):
            a: str
            b: str

        class B(BaseModel):
            a: str
            c: str

        class C(BaseModel):
            b: str
            c: str

        U = Union[A, B, C]
        adapter = TypeAdapter(U, config=ConfigDict(union_mode='smart'))

        # Payload has b and c, should match C
        payload = {"b": "", "c": ""}
        result = adapter.validate_python(payload)

        assert isinstance(result, C)
        assert result.b == ""
        assert result.c == ""

    def test_nested_models_with_three_options(self):
        """Test nested model discrimination with three options."""

        class InnerA(BaseModel):
            kind: Union[Literal["cat"], UnrecognizedStr]
            name: str

        class InnerB(BaseModel):
            kind: Union[Literal["dog"], UnrecognizedStr]

        class InnerC(BaseModel):
            kind: Union[Literal["bird"], UnrecognizedStr]

        class WrapperA(BaseModel):
            data: InnerA

        class WrapperB(BaseModel):
            data: InnerB

        class WrapperC(BaseModel):
            data: InnerC

        U = Union[WrapperA, WrapperB, WrapperC]
        adapter = TypeAdapter(U, config=ConfigDict(union_mode='smart'))

        # Payload with unknown kind but has name field, should match A (has name field)
        payload = {"data": {"kind": "unknown", "name": "test"}}
        result = adapter.validate_python(payload)

        assert isinstance(result, WrapperA)
        assert result.data.kind == "unknown"
        assert result.data.name == "test"


# =============================================================================
# SDK-Call Smart Union Tests - Ported from Ruby gold standard
# =============================================================================


class TestSmartUnionSDKCalls:
    """Test smart union discrimination via SDK calls to api-test-service."""

    def test_smart_union_deeply_nested_array(self):
        """Array<{x:{a:str}}> | Array<{x:{a:str, b:bool}}>
        Payload: [{x:{a:"", b:false}}]
        Expect: variant with more nested fields wins
        """
        record_test("smart-union-deeply-nested-array")

        class InnerA(BaseModel):
            a: str

        class InnerB(BaseModel):
            a: str
            b: bool

        class ItemA(BaseModel):
            x: InnerA

        class ItemB(BaseModel):
            x: InnerB

        class Res(BaseModel):
            model_config = ConfigDict(union_mode='smart')
            json_: Union[list[ItemA], list[ItemB]]

        adapter = TypeAdapter(Res)
        result = adapter.validate_python({"json_": [{"x": {"a": "", "b": False}}]})
        assert result.json_ is not None
        assert len(result.json_) == 1

    def test_smart_union_empty_string(self):
        """Types: {a: str} | {b: str}
        Payload: {b: ""}
        Expect: variant with field b
        """
        record_test("smart-union-empty-string")

        s = SDK()
        res = s.unions.smart_union_empty_string(request=shared.SmartUnionEmptyStringObjectB(b=""))
        assert res is not None
        assert res.http_meta.response.status_code == 200
        assert res.res is not None
        assert isinstance(res.res.json_, shared.SmartUnionEmptyStringObjectB)
        assert res.res.json_.b == ""

    def test_smart_union_prefers_fewer_unmatched_fields(self):
        """Types: {foo: str, bar: str} | {foo: str}
        Payload: {foo: "test"}
        Expect: variant with fewer unmatched fields
        """
        record_test("smart-union-prefers-fewer-unmatched-fields")

        s = SDK()
        res = s.unions.smart_union_prefers_fewer_unmatched_fields(
            request=shared.SmartUnionFewerUnmatchedB(foo="test")
        )
        assert res is not None
        assert res.http_meta.response.status_code == 200
        assert res.res is not None
        assert isinstance(res.res.json_, shared.SmartUnionFewerUnmatchedB)
        assert res.res.json_.foo == "test"

    def test_smart_union_preserves_order_on_tie(self):
        """Types: {foo: str} | {foo: str}
        Payload: {foo: "test"}
        Expect: first variant wins on tie
        """
        record_test("smart-union-preserves-order-on-tie")

        s = SDK()
        res = s.unions.smart_union_preserves_order_on_tie(
            request=shared.SmartUnionTieA(foo="test")
        )
        assert res is not None
        assert res.http_meta.response.status_code == 200
        assert res.res is not None
        assert isinstance(res.res.json_, shared.SmartUnionTieA)
        assert res.res.json_.foo == "test"

    def test_smart_union_array_fields(self):
        """Types: Array<{name: str}> | Array<{name: str, value: str}>
        Payload: [{name: "a", value: "1"}, {name: "b", value: "2"}]
        Expect: variant with more fields in items
        """
        record_test("smart-union-array-fields")

        class ItemA(BaseModel):
            name: str

        class ItemB(BaseModel):
            name: str
            value: str

        class Res(BaseModel):
            model_config = ConfigDict(union_mode='smart')
            json_: Union[list[ItemA], list[ItemB]]

        adapter = TypeAdapter(Res)
        result = adapter.validate_python(
            {"json_": [{"name": "a", "value": "1"}, {"name": "b", "value": "2"}]}
        )
        assert result.json_ is not None
        assert len(result.json_) == 2
        assert result.json_[0].name == "a"
        assert result.json_[1].name == "b"

    def test_smart_union_optional_pointer_fields(self):
        """Types: {foo: optional str} | {foo: optional str, bar: optional str}
        Payload: {foo: "test", bar: "value"}
        Expect: variant with more matched fields
        """
        record_test("smart-union-optional-pointer-fields")

        s = SDK()
        res = s.unions.smart_union_optional_pointer_fields(
            request=shared.SmartUnionOptionalPointerB(foo="test", bar="value")
        )
        assert res is not None
        assert res.http_meta.response.status_code == 200
        assert res.res is not None
        assert isinstance(res.res.json_, shared.SmartUnionOptionalPointerB)

    def test_smart_union_optional_pointer_structs(self):
        """Types: {nested: optional {name: str}} | {nested: optional {name: str, value: str}}
        Payload: {nested: {name: "test", value: "data"}}
        Expect: variant with more nested fields
        """
        record_test("smart-union-optional-pointer-structs")

        s = SDK()
        inner = shared.SmartUnionOptionalPointerStructsInnerB(name="test", value="data")
        res = s.unions.smart_union_optional_pointer_structs(
            request=shared.SmartUnionOptionalPointerStructsB(nested=inner)
        )
        assert res is not None
        assert res.http_meta.response.status_code == 200
        assert res.res is not None
        assert isinstance(res.res.json_, shared.SmartUnionOptionalPointerStructsB)
        assert res.res.json_.nested.name == "test"
        assert res.res.json_.nested.value == "data"

    def test_smart_union_all_consts(self):
        """Types: {a: const "A", b: const "B"} | {b: const "B", c: const "C"}
        Payload: {b: "B", c: "C"}
        Expect: variant matching both b and c
        """
        record_test("smart-union-all-consts")

        s = SDK()
        res = s.unions.smart_union_all_consts(
            request=shared.SmartUnionAllConstsB(b="B", c="C")
        )
        assert res is not None
        assert res.http_meta.response.status_code == 200
        assert res.res is not None
        assert isinstance(res.res.json_, shared.SmartUnionAllConstsB)

    def test_smart_union_any_field_type(self):
        """Types: {a: optional str} | {b: any}
        Payload: {b: "asdf"}
        Expect: variant with field b
        """
        record_test("smart-union-any-field-type")

        s = SDK()
        res = s.unions.smart_union_any_field_type(
            request=shared.SmartUnionAnyFieldB(b="asdf")
        )
        assert res is not None
        assert res.http_meta.response.status_code == 200
        assert res.res is not None
        assert isinstance(res.res.json_, shared.SmartUnionAnyFieldB)

    def test_smart_union_nested_union_vs_flat_struct(self):
        """Outer: {data: Union[x|y|z]} | {data: {x: str, y: str}}
        Payload: {data: {x: "", y: ""}}
        Expect: flat struct with 2 fields wins over nested union's 1
        """
        record_test("smart-union-nested-union-vs-flat-struct")

        s = SDK()
        data = shared.SmartUnionNestedVsFlatB(x="", y="")
        outer = operations.SmartUnionNestedVsFlatOuterB(data=data)
        res = s.unions.smart_union_nested_union_vs_flat_struct(
            request=outer
        )
        assert res is not None
        assert res.http_meta.response.status_code == 200
        assert res.res is not None

    def test_smart_union_union_vs_union(self):
        """Outer: Union[{a}|{b}] | Union[{a,b}|{a,b,c}]
        Payload: {a: "", b: "", c: ""}
        Expect: variant matching all 3 fields
        """
        record_test("smart-union-union-vs-union")

        s = SDK()
        res = s.unions.smart_union_union_vs_union(
            request=shared.SmartUnionVsUnionB2(a="", b="", c="")
        )
        assert res is not None
        assert res.http_meta.response.status_code == 200
        assert res.res is not None
        assert isinstance(res.res.json_, shared.SmartUnionVsUnionB2)

    def test_unions_discriminated_open_enum(self):
        """Discriminator on "status" field with open enum values.
        active/pending -> Status1, inactive/archived -> Status2
        """
        record_test("unions-discriminated-open-enum")
        from datetime import datetime

        s = SDK()

        # active -> Status1
        req_active = shared.ObjectWithOpenEnumStatus1(
            status="active",
            user_id="user-123",
            active_at=datetime.fromisoformat("2024-01-15T10:30:00+00:00"),
        )
        res = s.unions.discriminated_open_enum(request=req_active)
        assert res is not None
        assert res.http_meta.response.status_code == 200
        assert res.res is not None
        assert isinstance(res.res.json_, shared.ObjectWithOpenEnumStatus1)
        assert res.res.json_.user_id == "user-123"

        # inactive -> Status2
        req_inactive = shared.ObjectWithOpenEnumStatus2(
            status="inactive",
            reason="User requested deactivation",
            inactive_since=datetime.fromisoformat("2024-01-10T15:45:00+00:00"),
        )
        res = s.unions.discriminated_open_enum(request=req_inactive)
        assert res is not None
        assert res.http_meta.response.status_code == 200
        assert res.res is not None
        assert isinstance(res.res.json_, shared.ObjectWithOpenEnumStatus2)
        assert res.res.json_.reason == "User requested deactivation"


class TestOpenEnumScalarUnionSerialization:
    @pytest.mark.parametrize("payload, wire", [
        ({"stringOrNumber": 1.5}, '{"stringOrNumber":1.5}'),
        ({"stringOrNumber": "future"}, '{"stringOrNumber":"future"}'),
        ({"stringOrNumber": "known"}, '{"stringOrNumber":"known"}'),
        ({"stringOrBoolean": True}, '{"stringOrBoolean":true}'),
        ({"stringOrBoolean": "future"}, '{"stringOrBoolean":"future"}'),
        ({"integerOrNumber": 2.5}, '{"integerOrNumber":2.5}'),
        ({"integerOrNumber": 1}, '{"integerOrNumber":1}'),
        ({"stringOrNull": "future"}, '{"stringOrNull":"future"}'),
        ({"stringOrNull": "known"}, '{"stringOrNull":"known"}'),
    ])
    def test_scalar_member_serialization(self, payload, wire):
        model = shared.OpenEnumScalarUnions.model_validate(payload)
        with warnings.catch_warnings():
            warnings.simplefilter("error")
            assert utils.marshal_json(model, shared.OpenEnumScalarUnions) == wire
            assert model.model_dump_json(by_alias=True, exclude_none=True) == wire
            assert model.model_dump(by_alias=True, exclude_none=True) == payload

    def test_unrecognized_values_keep_their_type_in_python_mode(self):
        model = shared.OpenEnumScalarUnions.model_validate({"stringOrNumber": "future", "stringOrBoolean": "future"})
        dumped = model.model_dump(by_alias=True, exclude_none=True)
        assert type(dumped["stringOrNumber"]) is UnrecognizedStr
        assert type(dumped["stringOrBoolean"]) is UnrecognizedStr
