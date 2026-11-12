from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.automation_duration_unit import AutomationDurationUnit






T = TypeVar("T", bound="AutomationDuration")



@_attrs_define
class AutomationDuration:
    """ 
        Attributes:
            amount (int):
            unit (AutomationDurationUnit):
     """

    amount: int
    unit: AutomationDurationUnit





    def to_dict(self) -> dict[str, Any]:
        amount = self.amount

        unit = self.unit.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "amount": amount,
            "unit": unit,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        amount = d.pop("amount")

        unit = AutomationDurationUnit(d.pop("unit"))




        automation_duration = cls(
            amount=amount,
            unit=unit,
        )

        return automation_duration

