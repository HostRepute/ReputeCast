from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ComposerNameVersion")



@_attrs_define
class ComposerNameVersion:
    """ 
        Attributes:
            name (str): The version's name; empty removes it
     """

    name: str





    def to_dict(self) -> dict[str, Any]:
        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        composer_name_version = cls(
            name=name,
        )

        return composer_name_version

