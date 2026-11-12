from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.accessibility_policy_input_on_serious import AccessibilityPolicyInputOnSerious






T = TypeVar("T", bound="AccessibilityPolicyInput")



@_attrs_define
class AccessibilityPolicyInput:
    """ 
        Attributes:
            min_text_px (int):
            on_serious (AccessibilityPolicyInputOnSerious):
     """

    min_text_px: int
    on_serious: AccessibilityPolicyInputOnSerious





    def to_dict(self) -> dict[str, Any]:
        min_text_px = self.min_text_px

        on_serious = self.on_serious.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "minTextPx": min_text_px,
            "onSerious": on_serious,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        min_text_px = d.pop("minTextPx")

        on_serious = AccessibilityPolicyInputOnSerious(d.pop("onSerious"))




        accessibility_policy_input = cls(
            min_text_px=min_text_px,
            on_serious=on_serious,
        )

        return accessibility_policy_input

