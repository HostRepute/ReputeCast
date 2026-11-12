from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PlacementVariant")



@_attrs_define
class PlacementVariant:
    """ 
        Attributes:
            key (str):
            subject (str):
     """

    key: str
    subject: str





    def to_dict(self) -> dict[str, Any]:
        key = self.key

        subject = self.subject


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "subject": subject,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = d.pop("key")

        subject = d.pop("subject")

        placement_variant = cls(
            key=key,
            subject=subject,
        )

        return placement_variant

