from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="TransactionalMakeOptional")



@_attrs_define
class TransactionalMakeOptional:
    """ 
        Attributes:
            default (str):
            name (str):
     """

    default: str
    name: str





    def to_dict(self) -> dict[str, Any]:
        default = self.default

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "default": default,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        default = d.pop("default")

        name = d.pop("name")

        transactional_make_optional = cls(
            default=default,
            name=name,
        )

        return transactional_make_optional

