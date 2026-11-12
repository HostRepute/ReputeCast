from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WarmupHoldDecisionResult")



@_attrs_define
class WarmupHoldDecisionResult:
    """ 
        Attributes:
            action (str):
            cancelled (int):
            moved (int):
     """

    action: str
    cancelled: int
    moved: int





    def to_dict(self) -> dict[str, Any]:
        action = self.action

        cancelled = self.cancelled

        moved = self.moved


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "action": action,
            "cancelled": cancelled,
            "moved": moved,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        action = d.pop("action")

        cancelled = d.pop("cancelled")

        moved = d.pop("moved")

        warmup_hold_decision_result = cls(
            action=action,
            cancelled=cancelled,
            moved=moved,
        )

        return warmup_hold_decision_result

