from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="BlockChange")



@_attrs_define
class BlockChange:
    """ 
        Attributes:
            at (datetime.datetime):
            by (str):
            version (int):
     """

    at: datetime.datetime
    by: str
    version: int





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        by = self.by

        version = self.version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "by": by,
            "version": version,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        by = d.pop("by")

        version = d.pop("version")

        block_change = cls(
            at=at,
            by=by,
            version=version,
        )

        return block_change

