from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WebsiteCommentPost")



@_attrs_define
class WebsiteCommentPost:
    """ 
        Attributes:
            comments (int):
            id (str):
            title (str):
     """

    comments: int
    id: str
    title: str





    def to_dict(self) -> dict[str, Any]:
        comments = self.comments

        id = self.id

        title = self.title


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "comments": comments,
            "id": id,
            "title": title,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        comments = d.pop("comments")

        id = d.pop("id")

        title = d.pop("title")

        website_comment_post = cls(
            comments=comments,
            id=id,
            title=title,
        )

        return website_comment_post

