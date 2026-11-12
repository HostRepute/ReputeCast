from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="AccessibilityIssueVersion")



@_attrs_define
class AccessibilityIssueVersion:
    """ 
        Attributes:
            lang (str):
            suggestion (str | Unset):
            text (str | Unset):
     """

    lang: str
    suggestion: str | Unset = UNSET
    text: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        lang = self.lang

        suggestion = self.suggestion

        text = self.text


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "lang": lang,
        })
        if suggestion is not UNSET:
            field_dict["suggestion"] = suggestion
        if text is not UNSET:
            field_dict["text"] = text

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        lang = d.pop("lang")

        suggestion = d.pop("suggestion", UNSET)

        text = d.pop("text", UNSET)

        accessibility_issue_version = cls(
            lang=lang,
            suggestion=suggestion,
            text=text,
        )

        return accessibility_issue_version

