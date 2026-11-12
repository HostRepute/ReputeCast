from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="DigestPreview")



@_attrs_define
class DigestPreview:
    """ 
        Attributes:
            empty (bool):
            html (str):
            number (int):
            posts (int):
            preheader (str):
            subject (str):
     """

    empty: bool
    html: str
    number: int
    posts: int
    preheader: str
    subject: str





    def to_dict(self) -> dict[str, Any]:
        empty = self.empty

        html = self.html

        number = self.number

        posts = self.posts

        preheader = self.preheader

        subject = self.subject


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "empty": empty,
            "html": html,
            "number": number,
            "posts": posts,
            "preheader": preheader,
            "subject": subject,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        empty = d.pop("empty")

        html = d.pop("html")

        number = d.pop("number")

        posts = d.pop("posts")

        preheader = d.pop("preheader")

        subject = d.pop("subject")

        digest_preview = cls(
            empty=empty,
            html=html,
            number=number,
            posts=posts,
            preheader=preheader,
            subject=subject,
        )

        return digest_preview

