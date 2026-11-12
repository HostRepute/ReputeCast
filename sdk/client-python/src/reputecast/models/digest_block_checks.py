from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="DigestBlockChecks")



@_attrs_define
class DigestBlockChecks:
    """ 
        Attributes:
            alt_from_title (int):
            bytes_ (int):
            fallback (int):
            fit_posts (int):
            images (int):
            limit (int):
            links (int):
            max_bytes (int):
            max_posts (int):
     """

    alt_from_title: int
    bytes_: int
    fallback: int
    fit_posts: int
    images: int
    limit: int
    links: int
    max_bytes: int
    max_posts: int





    def to_dict(self) -> dict[str, Any]:
        alt_from_title = self.alt_from_title

        bytes_ = self.bytes_

        fallback = self.fallback

        fit_posts = self.fit_posts

        images = self.images

        limit = self.limit

        links = self.links

        max_bytes = self.max_bytes

        max_posts = self.max_posts


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "altFromTitle": alt_from_title,
            "bytes": bytes_,
            "fallback": fallback,
            "fitPosts": fit_posts,
            "images": images,
            "limit": limit,
            "links": links,
            "maxBytes": max_bytes,
            "maxPosts": max_posts,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        alt_from_title = d.pop("altFromTitle")

        bytes_ = d.pop("bytes")

        fallback = d.pop("fallback")

        fit_posts = d.pop("fitPosts")

        images = d.pop("images")

        limit = d.pop("limit")

        links = d.pop("links")

        max_bytes = d.pop("maxBytes")

        max_posts = d.pop("maxPosts")

        digest_block_checks = cls(
            alt_from_title=alt_from_title,
            bytes_=bytes_,
            fallback=fallback,
            fit_posts=fit_posts,
            images=images,
            limit=limit,
            links=links,
            max_bytes=max_bytes,
            max_posts=max_posts,
        )

        return digest_block_checks

