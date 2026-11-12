from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="SyncedInput")



@_attrs_define
class SyncedInput:
    """ 
        Attributes:
            mjml (str): One mj-section or mj-wrapper
            name (str | Unset): A named block's new name
     """

    mjml: str
    name: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        mjml = self.mjml

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "mjml": mjml,
        })
        if name is not UNSET:
            field_dict["name"] = name

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        mjml = d.pop("mjml")

        name = d.pop("name", UNSET)

        synced_input = cls(
            mjml=mjml,
            name=name,
        )

        return synced_input

