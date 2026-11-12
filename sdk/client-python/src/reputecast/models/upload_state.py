from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="UploadState")



@_attrs_define
class UploadState:
    """ 
        Attributes:
            chunk_bytes (int):
            id (str):
            received (int):
            size (int):
     """

    chunk_bytes: int
    id: str
    received: int
    size: int





    def to_dict(self) -> dict[str, Any]:
        chunk_bytes = self.chunk_bytes

        id = self.id

        received = self.received

        size = self.size


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "chunkBytes": chunk_bytes,
            "id": id,
            "received": received,
            "size": size,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        chunk_bytes = d.pop("chunkBytes")

        id = d.pop("id")

        received = d.pop("received")

        size = d.pop("size")

        upload_state = cls(
            chunk_bytes=chunk_bytes,
            id=id,
            received=received,
            size=size,
        )

        return upload_state

