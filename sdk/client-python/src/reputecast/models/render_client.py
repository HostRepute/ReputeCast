from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="RenderClient")



@_attrs_define
class RenderClient:
    """ 
        Attributes:
            default (bool):
            id (str):
            name (str):
     """

    default: bool
    id: str
    name: str





    def to_dict(self) -> dict[str, Any]:
        default = self.default

        id = self.id

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "default": default,
            "id": id,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        default = d.pop("default")

        id = d.pop("id")

        name = d.pop("name")

        render_client = cls(
            default=default,
            id=id,
            name=name,
        )

        return render_client

