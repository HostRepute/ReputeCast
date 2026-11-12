from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ComposerComment")



@_attrs_define
class ComposerComment:
    """ 
        Attributes:
            author (str):
            body (str):
            created_at (datetime.datetime):
            id (str):
            mentions (list[str]): Members it notified (user ids)
            reviewer (bool):
            version (int): The campaign version it was written on
            author_color (str | Unset):
            author_id (str | Unset):
     """

    author: str
    body: str
    created_at: datetime.datetime
    id: str
    mentions: list[str]
    reviewer: bool
    version: int
    author_color: str | Unset = UNSET
    author_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        author = self.author

        body = self.body

        created_at = self.created_at.isoformat()

        id = self.id

        mentions = self.mentions



        reviewer = self.reviewer

        version = self.version

        author_color = self.author_color

        author_id = self.author_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "author": author,
            "body": body,
            "createdAt": created_at,
            "id": id,
            "mentions": mentions,
            "reviewer": reviewer,
            "version": version,
        })
        if author_color is not UNSET:
            field_dict["authorColor"] = author_color
        if author_id is not UNSET:
            field_dict["authorId"] = author_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        author = d.pop("author")

        body = d.pop("body")

        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        id = d.pop("id")

        mentions = cast(list[str], d.pop("mentions"))


        reviewer = d.pop("reviewer")

        version = d.pop("version")

        author_color = d.pop("authorColor", UNSET)

        author_id = d.pop("authorId", UNSET)

        composer_comment = cls(
            author=author,
            body=body,
            created_at=created_at,
            id=id,
            mentions=mentions,
            reviewer=reviewer,
            version=version,
            author_color=author_color,
            author_id=author_id,
        )

        return composer_comment

