from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="SyncedCreateInput")



@_attrs_define
class SyncedCreateInput:
    """ 
        Attributes:
            mjml (str): One mj-section or mj-wrapper, from the email it is saved from
            name (str): What the block is called in the blocks panel and the brand kit
     """

    mjml: str
    name: str





    def to_dict(self) -> dict[str, Any]:
        mjml = self.mjml

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "mjml": mjml,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        mjml = d.pop("mjml")

        name = d.pop("name")

        synced_create_input = cls(
            mjml=mjml,
            name=name,
        )

        return synced_create_input

