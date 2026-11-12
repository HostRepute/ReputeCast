from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.accessibility_color_suggestion_reason import AccessibilityColorSuggestionReason






T = TypeVar("T", bound="AccessibilityColorSuggestion")



@_attrs_define
class AccessibilityColorSuggestion:
    """ 
        Attributes:
            color (str):  Example: #0F766E.
            ratio (float): Contrast ratio, rounded down to one decimal
            reason (AccessibilityColorSuggestionReason):
     """

    color: str
    ratio: float
    reason: AccessibilityColorSuggestionReason





    def to_dict(self) -> dict[str, Any]:
        color = self.color

        ratio = self.ratio

        reason = self.reason.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "color": color,
            "ratio": ratio,
            "reason": reason,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        color = d.pop("color")

        ratio = d.pop("ratio")

        reason = AccessibilityColorSuggestionReason(d.pop("reason"))




        accessibility_color_suggestion = cls(
            color=color,
            ratio=ratio,
            reason=reason,
        )

        return accessibility_color_suggestion

