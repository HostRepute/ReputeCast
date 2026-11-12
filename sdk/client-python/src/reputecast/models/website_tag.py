from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="WebsiteTag")



@_attrs_define
class WebsiteTag:
    """ 
        Attributes:
            description (str):
            feed (str):
            id (str):
            in_nav (bool):
            name (str):
            posts (int):
            slug (str):
            unpublished (int):
            url (str):
            list_ (str | Unset):
     """

    description: str
    feed: str
    id: str
    in_nav: bool
    name: str
    posts: int
    slug: str
    unpublished: int
    url: str
    list_: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        description = self.description

        feed = self.feed

        id = self.id

        in_nav = self.in_nav

        name = self.name

        posts = self.posts

        slug = self.slug

        unpublished = self.unpublished

        url = self.url

        list_ = self.list_


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "description": description,
            "feed": feed,
            "id": id,
            "inNav": in_nav,
            "name": name,
            "posts": posts,
            "slug": slug,
            "unpublished": unpublished,
            "url": url,
        })
        if list_ is not UNSET:
            field_dict["list"] = list_

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        description = d.pop("description")

        feed = d.pop("feed")

        id = d.pop("id")

        in_nav = d.pop("inNav")

        name = d.pop("name")

        posts = d.pop("posts")

        slug = d.pop("slug")

        unpublished = d.pop("unpublished")

        url = d.pop("url")

        list_ = d.pop("list", UNSET)

        website_tag = cls(
            description=description,
            feed=feed,
            id=id,
            in_nav=in_nav,
            name=name,
            posts=posts,
            slug=slug,
            unpublished=unpublished,
            url=url,
            list_=list_,
        )

        return website_tag

