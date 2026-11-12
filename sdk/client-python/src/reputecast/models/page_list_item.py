from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.page_list_item_status import PageListItemStatus
from ..models.page_list_item_template import PageListItemTemplate
from ..models.page_list_item_visibility import PageListItemVisibility
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.page_person import PagePerson
  from ..models.page_result import PageResult





T = TypeVar("T", bound="PageListItem")



@_attrs_define
class PageListItem:
    """ 
        Attributes:
            changes (bool):
            edited_at (datetime.datetime):
            id (str):
            name (str):
            path (str):
            result (PageResult):
            slug (str):
            status (PageListItemStatus):
            template (PageListItemTemplate):
            url (str):
            views (int):
            visibility (PageListItemVisibility):
            countdown_ended (datetime.datetime | Unset):
            edited_by (PagePerson | Unset):
            first_published_at (datetime.datetime | Unset):
            publish_at (datetime.datetime | Unset):
            published_at (datetime.datetime | Unset):
            unpublished_at (datetime.datetime | Unset):
     """

    changes: bool
    edited_at: datetime.datetime
    id: str
    name: str
    path: str
    result: PageResult
    slug: str
    status: PageListItemStatus
    template: PageListItemTemplate
    url: str
    views: int
    visibility: PageListItemVisibility
    countdown_ended: datetime.datetime | Unset = UNSET
    edited_by: PagePerson | Unset = UNSET
    first_published_at: datetime.datetime | Unset = UNSET
    publish_at: datetime.datetime | Unset = UNSET
    published_at: datetime.datetime | Unset = UNSET
    unpublished_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.page_person import PagePerson # noqa: PLC0415
        from ..models.page_result import PageResult # noqa: PLC0415
        changes = self.changes

        edited_at = self.edited_at.isoformat()

        id = self.id

        name = self.name

        path = self.path

        result = self.result.to_dict()

        slug = self.slug

        status = self.status.value

        template = self.template.value

        url = self.url

        views = self.views

        visibility = self.visibility.value

        countdown_ended: str | Unset = UNSET
        if not isinstance(self.countdown_ended, Unset):
            countdown_ended = self.countdown_ended.isoformat()

        edited_by: dict[str, Any] | Unset = UNSET
        if not isinstance(self.edited_by, Unset):
            edited_by = self.edited_by.to_dict()

        first_published_at: str | Unset = UNSET
        if not isinstance(self.first_published_at, Unset):
            first_published_at = self.first_published_at.isoformat()

        publish_at: str | Unset = UNSET
        if not isinstance(self.publish_at, Unset):
            publish_at = self.publish_at.isoformat()

        published_at: str | Unset = UNSET
        if not isinstance(self.published_at, Unset):
            published_at = self.published_at.isoformat()

        unpublished_at: str | Unset = UNSET
        if not isinstance(self.unpublished_at, Unset):
            unpublished_at = self.unpublished_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "changes": changes,
            "editedAt": edited_at,
            "id": id,
            "name": name,
            "path": path,
            "result": result,
            "slug": slug,
            "status": status,
            "template": template,
            "url": url,
            "views": views,
            "visibility": visibility,
        })
        if countdown_ended is not UNSET:
            field_dict["countdownEnded"] = countdown_ended
        if edited_by is not UNSET:
            field_dict["editedBy"] = edited_by
        if first_published_at is not UNSET:
            field_dict["firstPublishedAt"] = first_published_at
        if publish_at is not UNSET:
            field_dict["publishAt"] = publish_at
        if published_at is not UNSET:
            field_dict["publishedAt"] = published_at
        if unpublished_at is not UNSET:
            field_dict["unpublishedAt"] = unpublished_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.page_person import PagePerson # noqa: PLC0415
        from ..models.page_result import PageResult # noqa: PLC0415
        d = dict(src_dict)
        changes = d.pop("changes")

        edited_at = datetime.datetime.fromisoformat(d.pop("editedAt"))




        id = d.pop("id")

        name = d.pop("name")

        path = d.pop("path")

        result = PageResult.from_dict(d.pop("result"))




        slug = d.pop("slug")

        status = PageListItemStatus(d.pop("status"))




        template = PageListItemTemplate(d.pop("template"))




        url = d.pop("url")

        views = d.pop("views")

        visibility = PageListItemVisibility(d.pop("visibility"))




        _countdown_ended = d.pop("countdownEnded", UNSET)
        countdown_ended: datetime.datetime | Unset
        if isinstance(_countdown_ended,  Unset):
            countdown_ended = UNSET
        else:
            countdown_ended = datetime.datetime.fromisoformat(_countdown_ended)




        _edited_by = d.pop("editedBy", UNSET)
        edited_by: PagePerson | Unset
        if isinstance(_edited_by,  Unset):
            edited_by = UNSET
        else:
            edited_by = PagePerson.from_dict(_edited_by)




        _first_published_at = d.pop("firstPublishedAt", UNSET)
        first_published_at: datetime.datetime | Unset
        if isinstance(_first_published_at,  Unset):
            first_published_at = UNSET
        else:
            first_published_at = datetime.datetime.fromisoformat(_first_published_at)




        _publish_at = d.pop("publishAt", UNSET)
        publish_at: datetime.datetime | Unset
        if isinstance(_publish_at,  Unset):
            publish_at = UNSET
        else:
            publish_at = datetime.datetime.fromisoformat(_publish_at)




        _published_at = d.pop("publishedAt", UNSET)
        published_at: datetime.datetime | Unset
        if isinstance(_published_at,  Unset):
            published_at = UNSET
        else:
            published_at = datetime.datetime.fromisoformat(_published_at)




        _unpublished_at = d.pop("unpublishedAt", UNSET)
        unpublished_at: datetime.datetime | Unset
        if isinstance(_unpublished_at,  Unset):
            unpublished_at = UNSET
        else:
            unpublished_at = datetime.datetime.fromisoformat(_unpublished_at)




        page_list_item = cls(
            changes=changes,
            edited_at=edited_at,
            id=id,
            name=name,
            path=path,
            result=result,
            slug=slug,
            status=status,
            template=template,
            url=url,
            views=views,
            visibility=visibility,
            countdown_ended=countdown_ended,
            edited_by=edited_by,
            first_published_at=first_published_at,
            publish_at=publish_at,
            published_at=published_at,
            unpublished_at=unpublished_at,
        )

        return page_list_item

