from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="MergedSection")



@_attrs_define
class MergedSection:
    """ 
        Attributes:
            index (int):
            label (str):
            mjml (str):
     """

    index: int
    label: str
    mjml: str





    def to_dict(self) -> dict[str, Any]:
        index = self.index

        label = self.label

        mjml = self.mjml


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "index": index,
            "label": label,
            "mjml": mjml,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        index = d.pop("index")

        label = d.pop("label")

        mjml = d.pop("mjml")

        merged_section = cls(
            index=index,
            label=label,
            mjml=mjml,
        )

        return merged_section

