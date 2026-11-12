from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from uuid import UUID






T = TypeVar("T", bound="UploadStart")



@_attrs_define
class UploadStart:
    """ 
        Attributes:
            name (str):
            size (int):
            replaces (UUID | Unset):
     """

    name: str
    size: int
    replaces: UUID | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        name = self.name

        size = self.size

        replaces: str | Unset = UNSET
        if not isinstance(self.replaces, Unset):
            replaces = str(self.replaces)


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "name": name,
            "size": size,
        })
        if replaces is not UNSET:
            field_dict["replaces"] = replaces

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        size = d.pop("size")

        _replaces = d.pop("replaces", UNSET)
        replaces: UUID | Unset
        if isinstance(_replaces,  Unset):
            replaces = UNSET
        else:
            replaces = UUID(_replaces)




        upload_start = cls(
            name=name,
            size=size,
            replaces=replaces,
        )

        return upload_start

