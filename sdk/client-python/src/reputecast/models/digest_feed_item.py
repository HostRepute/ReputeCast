from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.digest_feed_item_left_out_by import DigestFeedItemLeftOutBy
from ..models.digest_feed_item_status import DigestFeedItemStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="DigestFeedItem")



@_attrs_define
class DigestFeedItem:
    """ 
        Attributes:
            alt_from_title (bool):
            author (str):
            categories (list[str]):
            excerpt (str):
            fallback (bool):
            first_seen_at (datetime.datetime):
            first_seen_via (str):
            id (str):
            status (DigestFeedItemStatus):
            tags (list[str]):
            title (str):
            url (str):
            image (str | Unset):
            image_alt (str | Unset):
            left_out_by (DigestFeedItemLeftOutBy | Unset):
            published (datetime.datetime | Unset):
            sent_in (int | Unset):
     """

    alt_from_title: bool
    author: str
    categories: list[str]
    excerpt: str
    fallback: bool
    first_seen_at: datetime.datetime
    first_seen_via: str
    id: str
    status: DigestFeedItemStatus
    tags: list[str]
    title: str
    url: str
    image: str | Unset = UNSET
    image_alt: str | Unset = UNSET
    left_out_by: DigestFeedItemLeftOutBy | Unset = UNSET
    published: datetime.datetime | Unset = UNSET
    sent_in: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        alt_from_title = self.alt_from_title

        author = self.author

        categories = self.categories



        excerpt = self.excerpt

        fallback = self.fallback

        first_seen_at = self.first_seen_at.isoformat()

        first_seen_via = self.first_seen_via

        id = self.id

        status = self.status.value

        tags = self.tags



        title = self.title

        url = self.url

        image = self.image

        image_alt = self.image_alt

        left_out_by: str | Unset = UNSET
        if not isinstance(self.left_out_by, Unset):
            left_out_by = self.left_out_by.value


        published: str | Unset = UNSET
        if not isinstance(self.published, Unset):
            published = self.published.isoformat()

        sent_in = self.sent_in


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "altFromTitle": alt_from_title,
            "author": author,
            "categories": categories,
            "excerpt": excerpt,
            "fallback": fallback,
            "firstSeenAt": first_seen_at,
            "firstSeenVia": first_seen_via,
            "id": id,
            "status": status,
            "tags": tags,
            "title": title,
            "url": url,
        })
        if image is not UNSET:
            field_dict["image"] = image
        if image_alt is not UNSET:
            field_dict["imageAlt"] = image_alt
        if left_out_by is not UNSET:
            field_dict["leftOutBy"] = left_out_by
        if published is not UNSET:
            field_dict["published"] = published
        if sent_in is not UNSET:
            field_dict["sentIn"] = sent_in

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        alt_from_title = d.pop("altFromTitle")

        author = d.pop("author")

        categories = cast(list[str], d.pop("categories"))


        excerpt = d.pop("excerpt")

        fallback = d.pop("fallback")

        first_seen_at = datetime.datetime.fromisoformat(d.pop("firstSeenAt"))




        first_seen_via = d.pop("firstSeenVia")

        id = d.pop("id")

        status = DigestFeedItemStatus(d.pop("status"))




        tags = cast(list[str], d.pop("tags"))


        title = d.pop("title")

        url = d.pop("url")

        image = d.pop("image", UNSET)

        image_alt = d.pop("imageAlt", UNSET)

        _left_out_by = d.pop("leftOutBy", UNSET)
        left_out_by: DigestFeedItemLeftOutBy | Unset
        if isinstance(_left_out_by,  Unset):
            left_out_by = UNSET
        else:
            left_out_by = DigestFeedItemLeftOutBy(_left_out_by)




        _published = d.pop("published", UNSET)
        published: datetime.datetime | Unset
        if isinstance(_published,  Unset):
            published = UNSET
        else:
            published = datetime.datetime.fromisoformat(_published)




        sent_in = d.pop("sentIn", UNSET)

        digest_feed_item = cls(
            alt_from_title=alt_from_title,
            author=author,
            categories=categories,
            excerpt=excerpt,
            fallback=fallback,
            first_seen_at=first_seen_at,
            first_seen_via=first_seen_via,
            id=id,
            status=status,
            tags=tags,
            title=title,
            url=url,
            image=image,
            image_alt=image_alt,
            left_out_by=left_out_by,
            published=published,
            sent_in=sent_in,
        )

        return digest_feed_item

