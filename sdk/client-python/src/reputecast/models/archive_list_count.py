from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ArchiveListCount")



@_attrs_define
class ArchiveListCount:
    """ 
        Attributes:
            issues (int):
            list_id (str):
            name (str):
     """

    issues: int
    list_id: str
    name: str





    def to_dict(self) -> dict[str, Any]:
        issues = self.issues

        list_id = self.list_id

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "issues": issues,
            "listId": list_id,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        issues = d.pop("issues")

        list_id = d.pop("listId")

        name = d.pop("name")

        archive_list_count = cls(
            issues=issues,
            list_id=list_id,
            name=name,
        )

        return archive_list_count

