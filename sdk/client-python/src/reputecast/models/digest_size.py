from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="DigestSize")



@_attrs_define
class DigestSize:
    """ 
        Attributes:
            bytes_ (int):
            posts (int):
     """

    bytes_: int
    posts: int





    def to_dict(self) -> dict[str, Any]:
        bytes_ = self.bytes_

        posts = self.posts


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "bytes": bytes_,
            "posts": posts,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        bytes_ = d.pop("bytes")

        posts = d.pop("posts")

        digest_size = cls(
            bytes_=bytes_,
            posts=posts,
        )

        return digest_size

