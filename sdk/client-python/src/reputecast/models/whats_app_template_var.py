from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="WhatsAppTemplateVar")



@_attrs_define
class WhatsAppTemplateVar:
    """ 
        Attributes:
            key (str):
            label (str):
            example (str | Unset):
     """

    key: str
    label: str
    example: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        key = self.key

        label = self.label

        example = self.example


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "label": label,
        })
        if example is not UNSET:
            field_dict["example"] = example

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = d.pop("key")

        label = d.pop("label")

        example = d.pop("example", UNSET)

        whats_app_template_var = cls(
            key=key,
            label=label,
            example=example,
        )

        return whats_app_template_var

