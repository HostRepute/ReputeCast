from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.website_post_counts import WebsitePostCounts
  from ..models.website_post_row import WebsitePostRow





T = TypeVar("T", bound="WebsitePostList")



@_attrs_define
class WebsitePostList:
    """ 
        Attributes:
            counts (WebsitePostCounts):
            from_ (datetime.datetime):
            page (int):
            page_size (int):
            posts (list[WebsitePostRow]):
            total (int):
            until (datetime.datetime):
     """

    counts: WebsitePostCounts
    from_: datetime.datetime
    page: int
    page_size: int
    posts: list[WebsitePostRow]
    total: int
    until: datetime.datetime





    def to_dict(self) -> dict[str, Any]:
        from ..models.website_post_counts import WebsitePostCounts # noqa: PLC0415
        from ..models.website_post_row import WebsitePostRow # noqa: PLC0415
        counts = self.counts.to_dict()

        from_ = self.from_.isoformat()

        page = self.page

        page_size = self.page_size

        posts = []
        for posts_item_data in self.posts:
            posts_item = posts_item_data.to_dict()
            posts.append(posts_item)



        total = self.total

        until = self.until.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "counts": counts,
            "from": from_,
            "page": page,
            "pageSize": page_size,
            "posts": posts,
            "total": total,
            "until": until,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.website_post_counts import WebsitePostCounts # noqa: PLC0415
        from ..models.website_post_row import WebsitePostRow # noqa: PLC0415
        d = dict(src_dict)
        counts = WebsitePostCounts.from_dict(d.pop("counts"))




        from_ = datetime.datetime.fromisoformat(d.pop("from"))




        page = d.pop("page")

        page_size = d.pop("pageSize")

        posts = []
        _posts = d.pop("posts")
        for posts_item_data in (_posts):
            posts_item = WebsitePostRow.from_dict(posts_item_data)



            posts.append(posts_item)


        total = d.pop("total")

        until = datetime.datetime.fromisoformat(d.pop("until"))




        website_post_list = cls(
            counts=counts,
            from_=from_,
            page=page,
            page_size=page_size,
            posts=posts,
            total=total,
            until=until,
        )

        return website_post_list

