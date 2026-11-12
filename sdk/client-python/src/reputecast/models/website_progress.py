from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.website_progress_pages import WebsiteProgressPages
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.website_progress_step import WebsiteProgressStep





T = TypeVar("T", bound="WebsiteProgress")



@_attrs_define
class WebsiteProgress:
    """ 
        Attributes:
            pages (WebsiteProgressPages):
            posts (WebsiteProgressStep):
            redirects (WebsiteProgressStep):
            tags (WebsiteProgressStep):
            finished_at (datetime.datetime | Unset):
            started_at (datetime.datetime | Unset):
     """

    pages: WebsiteProgressPages
    posts: WebsiteProgressStep
    redirects: WebsiteProgressStep
    tags: WebsiteProgressStep
    finished_at: datetime.datetime | Unset = UNSET
    started_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.website_progress_step import WebsiteProgressStep # noqa: PLC0415
        pages = self.pages.value

        posts = self.posts.to_dict()

        redirects = self.redirects.to_dict()

        tags = self.tags.to_dict()

        finished_at: str | Unset = UNSET
        if not isinstance(self.finished_at, Unset):
            finished_at = self.finished_at.isoformat()

        started_at: str | Unset = UNSET
        if not isinstance(self.started_at, Unset):
            started_at = self.started_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "pages": pages,
            "posts": posts,
            "redirects": redirects,
            "tags": tags,
        })
        if finished_at is not UNSET:
            field_dict["finishedAt"] = finished_at
        if started_at is not UNSET:
            field_dict["startedAt"] = started_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.website_progress_step import WebsiteProgressStep # noqa: PLC0415
        d = dict(src_dict)
        pages = WebsiteProgressPages(d.pop("pages"))




        posts = WebsiteProgressStep.from_dict(d.pop("posts"))




        redirects = WebsiteProgressStep.from_dict(d.pop("redirects"))




        tags = WebsiteProgressStep.from_dict(d.pop("tags"))




        _finished_at = d.pop("finishedAt", UNSET)
        finished_at: datetime.datetime | Unset
        if isinstance(_finished_at,  Unset):
            finished_at = UNSET
        else:
            finished_at = datetime.datetime.fromisoformat(_finished_at)




        _started_at = d.pop("startedAt", UNSET)
        started_at: datetime.datetime | Unset
        if isinstance(_started_at,  Unset):
            started_at = UNSET
        else:
            started_at = datetime.datetime.fromisoformat(_started_at)




        website_progress = cls(
            pages=pages,
            posts=posts,
            redirects=redirects,
            tags=tags,
            finished_at=finished_at,
            started_at=started_at,
        )

        return website_progress

