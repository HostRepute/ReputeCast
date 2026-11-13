from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="LeadAdsCheckbox")



@_attrs_define
class LeadAdsCheckbox:
    """ 
        Attributes:
            key (str):
            required (bool):
            text (str):
     """

    key: str
    required: bool
    text: str





    def to_dict(self) -> dict[str, Any]:
        key = self.key

        required = self.required

        text = self.text


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "required": required,
            "text": text,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = d.pop("key")

        required = d.pop("required")

        text = d.pop("text")

        lead_ads_checkbox = cls(
            key=key,
            required=required,
            text=text,
        )

        return lead_ads_checkbox

