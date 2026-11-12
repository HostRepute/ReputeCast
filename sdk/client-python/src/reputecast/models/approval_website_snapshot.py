from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.approval_website_snapshot_kind import ApprovalWebsiteSnapshotKind
from ..models.approval_website_snapshot_visibility import ApprovalWebsiteSnapshotVisibility
from ..models.approval_website_snapshot_when import ApprovalWebsiteSnapshotWhen
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ApprovalWebsiteSnapshot")



@_attrs_define
class ApprovalWebsiteSnapshot:
    """ 
        Attributes:
            excerpt (str):
            kind (ApprovalWebsiteSnapshotKind):
            path (str):
            post_id (str):
            published (bool):
            slug (str):
            title (str):
            visibility (ApprovalWebsiteSnapshotVisibility):
            visibility_label (str):
            when (ApprovalWebsiteSnapshotWhen):
            publish_at (datetime.datetime | Unset):
     """

    excerpt: str
    kind: ApprovalWebsiteSnapshotKind
    path: str
    post_id: str
    published: bool
    slug: str
    title: str
    visibility: ApprovalWebsiteSnapshotVisibility
    visibility_label: str
    when: ApprovalWebsiteSnapshotWhen
    publish_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        excerpt = self.excerpt

        kind = self.kind.value

        path = self.path

        post_id = self.post_id

        published = self.published

        slug = self.slug

        title = self.title

        visibility = self.visibility.value

        visibility_label = self.visibility_label

        when = self.when.value

        publish_at: str | Unset = UNSET
        if not isinstance(self.publish_at, Unset):
            publish_at = self.publish_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "excerpt": excerpt,
            "kind": kind,
            "path": path,
            "postId": post_id,
            "published": published,
            "slug": slug,
            "title": title,
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
        excerpt = d.pop("excerpt")

        kind = ApprovalWebsiteSnapshotKind(d.pop("kind"))




        path = d.pop("path")

        post_id = d.pop("postId")

        published = d.pop("published")

        slug = d.pop("slug")

        title = d.pop("title")

        visibility = ApprovalWebsiteSnapshotVisibility(d.pop("visibility"))




        visibility_label = d.pop("visibilityLabel")

        when = ApprovalWebsiteSnapshotWhen(d.pop("when"))




        _publish_at = d.pop("publishAt", UNSET)
        publish_at: datetime.datetime | Unset
        if isinstance(_publish_at,  Unset):
            publish_at = UNSET
        else:
            publish_at = datetime.datetime.fromisoformat(_publish_at)




        approval_website_snapshot = cls(
            excerpt=excerpt,
            kind=kind,
            path=path,
            post_id=post_id,
            published=published,
            slug=slug,
            title=title,
            visibility=visibility,
            visibility_label=visibility_label,
            when=when,
            publish_at=publish_at,
        )

        return approval_website_snapshot

