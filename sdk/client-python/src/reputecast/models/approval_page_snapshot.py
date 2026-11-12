from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.approval_page_snapshot_visibility import ApprovalPageSnapshotVisibility
from ..models.approval_page_snapshot_when import ApprovalPageSnapshotWhen
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ApprovalPageSnapshot")



@_attrs_define
class ApprovalPageSnapshot:
    """ 
        Attributes:
            name (str):
            page_id (str):
            path (str):
            published (bool):
            slug (str):
            template (str):
            visibility (ApprovalPageSnapshotVisibility):
            visibility_label (str):
            when (ApprovalPageSnapshotWhen):
            publish_at (datetime.datetime | Unset):
     """

    name: str
    page_id: str
    path: str
    published: bool
    slug: str
    template: str
    visibility: ApprovalPageSnapshotVisibility
    visibility_label: str
    when: ApprovalPageSnapshotWhen
    publish_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        name = self.name

        page_id = self.page_id

        path = self.path

        published = self.published

        slug = self.slug

        template = self.template

        visibility = self.visibility.value

        visibility_label = self.visibility_label

        when = self.when.value

        publish_at: str | Unset = UNSET
        if not isinstance(self.publish_at, Unset):
            publish_at = self.publish_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "name": name,
            "pageId": page_id,
            "path": path,
            "published": published,
            "slug": slug,
            "template": template,
            "visibility": visibility,
            "visibilityLabel": visibility_label,
            "when": when,
        })
        if publish_at is not UNSET:
            field_dict["publishAt"] = publish_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        page_id = d.pop("pageId")

        path = d.pop("path")

        published = d.pop("published")

        slug = d.pop("slug")

        template = d.pop("template")

        visibility = ApprovalPageSnapshotVisibility(d.pop("visibility"))




        visibility_label = d.pop("visibilityLabel")

        when = ApprovalPageSnapshotWhen(d.pop("when"))




        _publish_at = d.pop("publishAt", UNSET)
        publish_at: datetime.datetime | Unset
        if isinstance(_publish_at,  Unset):
            publish_at = UNSET
        else:
            publish_at = datetime.datetime.fromisoformat(_publish_at)




        approval_page_snapshot = cls(
            name=name,
            page_id=page_id,
            path=path,
            published=published,
            slug=slug,
            template=template,
            visibility=visibility,
            visibility_label=visibility_label,
            when=when,
            publish_at=publish_at,
        )

        return approval_page_snapshot

