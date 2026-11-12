from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="AutomationGoal")



@_attrs_define
class AutomationGoal:
    """ 
        Attributes:
            label (str):  Example: Clicked docs.
            percent (float | Unset): Contacts who reached the goal, % of those who entered in the window
     """

    label: str
    percent: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        label = self.label

        percent = self.percent


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "label": label,
        })
        if percent is not UNSET:
            field_dict["percent"] = percent

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        label = d.pop("label")

        percent = d.pop("percent", UNSET)

        automation_goal = cls(
            label=label,
            percent=percent,
        )

        return automation_goal

