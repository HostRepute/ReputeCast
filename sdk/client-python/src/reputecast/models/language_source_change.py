from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="LanguageSourceChange")



@_attrs_define
class LanguageSourceChange:
    """ 
        Attributes:
            at (datetime.datetime):
            blocks (list[str]):
            by (str):
            outdated (list[str]):
            version (int):
     """

    at: datetime.datetime
    blocks: list[str]
    by: str
    outdated: list[str]
    version: int





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        blocks = self.blocks



        by = self.by

        outdated = self.outdated



        version = self.version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "blocks": blocks,
            "by": by,
            "outdated": outdated,
            "version": version,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        blocks = cast(list[str], d.pop("blocks"))


        by = d.pop("by")

        outdated = cast(list[str], d.pop("outdated"))


        version = d.pop("version")

        language_source_change = cls(
            at=at,
            blocks=blocks,
            by=by,
            outdated=outdated,
            version=version,
        )

        return language_source_change

