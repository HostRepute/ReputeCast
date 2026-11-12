from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WebsiteArchiveList")



@_attrs_define
class WebsiteArchiveList:
    """ 
        Attributes:
            archived (bool):
            id (str):
            issues (int):
            name (str):
            public (bool):
     """

    archived: bool
    id: str
    issues: int
    name: str
    public: bool





    def to_dict(self) -> dict[str, Any]:
        archived = self.archived

        id = self.id

        issues = self.issues

        name = self.name

        public = self.public


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "archived": archived,
            "id": id,
            "issues": issues,
            "name": name,
            "public": public,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        archived = d.pop("archived")

        id = d.pop("id")

        issues = d.pop("issues")

        name = d.pop("name")

        public = d.pop("public")

        website_archive_list = cls(
            archived=archived,
            id=id,
            issues=issues,
            name=name,
            public=public,
        )

        return website_archive_list

