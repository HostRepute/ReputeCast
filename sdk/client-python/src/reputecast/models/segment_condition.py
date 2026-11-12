from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.segment_condition_unit import SegmentConditionUnit
from ..types import UNSET, Unset






T = TypeVar("T", bound="SegmentCondition")



@_attrs_define
class SegmentCondition:
    """ 
        Attributes:
            field (str):
            op (str):
            amount (int | Unset):
            to (str | Unset):
            unit (SegmentConditionUnit | Unset):
            value (str | Unset):
     """

    field: str
    op: str
    amount: int | Unset = UNSET
    to: str | Unset = UNSET
    unit: SegmentConditionUnit | Unset = UNSET
    value: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        field = self.field

        op = self.op

        amount = self.amount

        to = self.to

        unit: str | Unset = UNSET
        if not isinstance(self.unit, Unset):
            unit = self.unit.value


        value = self.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "field": field,
            "op": op,
        })
        if amount is not UNSET:
            field_dict["amount"] = amount
        if to is not UNSET:
            field_dict["to"] = to
        if unit is not UNSET:
            field_dict["unit"] = unit
        if value is not UNSET:
            field_dict["value"] = value

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        field = d.pop("field")

        op = d.pop("op")

        amount = d.pop("amount", UNSET)

        to = d.pop("to", UNSET)

        _unit = d.pop("unit", UNSET)
        unit: SegmentConditionUnit | Unset
        if isinstance(_unit,  Unset):
            unit = UNSET
        else:
            unit = SegmentConditionUnit(_unit)




        value = d.pop("value", UNSET)

        segment_condition = cls(
            field=field,
            op=op,
            amount=amount,
            to=to,
            unit=unit,
            value=value,
        )

        return segment_condition

