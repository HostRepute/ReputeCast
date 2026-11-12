from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="AccessibilityCheckRequest")



@_attrs_define
class AccessibilityCheckRequest:
    """ 
        Attributes:
            html (str | Unset):
            lang (str | Unset): The language version being edited (the source when empty)
            mjml (str | Unset):
     """

    html: str | Unset = UNSET
    lang: str | Unset = UNSET
    mjml: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        html = self.html

        lang = self.lang

        mjml = self.mjml


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if html is not UNSET:
            field_dict["html"] = html
        if lang is not UNSET:
            field_dict["lang"] = lang
        if mjml is not UNSET:
            field_dict["mjml"] = mjml

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        html = d.pop("html", UNSET)

        lang = d.pop("lang", UNSET)

        mjml = d.pop("mjml", UNSET)

        accessibility_check_request = cls(
            html=html,
            lang=lang,
            mjml=mjml,
        )

        return accessibility_check_request

