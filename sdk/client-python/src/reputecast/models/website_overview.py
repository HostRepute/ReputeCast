from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.website_overview_home_layout import WebsiteOverviewHomeLayout
from ..models.website_overview_status import WebsiteOverviewStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.website_address import WebsiteAddress
  from ..models.website_archive_facts import WebsiteArchiveFacts
  from ..models.website_helpers import WebsiteHelpers
  from ..models.website_members_facts import WebsiteMembersFacts
  from ..models.website_numbers import WebsiteNumbers
  from ..models.website_off_facts import WebsiteOffFacts
  from ..models.website_post_counts import WebsitePostCounts
  from ..models.website_progress import WebsiteProgress





T = TypeVar("T", bound="WebsiteOverview")



@_attrs_define
class WebsiteOverview:
    """ 
        Attributes:
            address (WebsiteAddress):
            archive (WebsiteArchiveFacts):
            helpers (WebsiteHelpers):
            home_layout (WebsiteOverviewHomeLayout):
            members (WebsiteMembersFacts):
            numbers (WebsiteNumbers):
            off (WebsiteOffFacts):
            posts (WebsitePostCounts):
            status (WebsiteOverviewStatus):
            title (str):
            updated_at (datetime.datetime):
            progress (WebsiteProgress | Unset):
            turned_on_at (datetime.datetime | Unset):
            turned_on_by (str | Unset):
            updated_by (str | Unset):
     """

    address: WebsiteAddress
    archive: WebsiteArchiveFacts
    helpers: WebsiteHelpers
    home_layout: WebsiteOverviewHomeLayout
    members: WebsiteMembersFacts
    numbers: WebsiteNumbers
    off: WebsiteOffFacts
    posts: WebsitePostCounts
    status: WebsiteOverviewStatus
    title: str
    updated_at: datetime.datetime
    progress: WebsiteProgress | Unset = UNSET
    turned_on_at: datetime.datetime | Unset = UNSET
    turned_on_by: str | Unset = UNSET
    updated_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.website_address import WebsiteAddress # noqa: PLC0415
        from ..models.website_archive_facts import WebsiteArchiveFacts # noqa: PLC0415
        from ..models.website_helpers import WebsiteHelpers # noqa: PLC0415
        from ..models.website_members_facts import WebsiteMembersFacts # noqa: PLC0415
        from ..models.website_numbers import WebsiteNumbers # noqa: PLC0415
        from ..models.website_off_facts import WebsiteOffFacts # noqa: PLC0415
        from ..models.website_post_counts import WebsitePostCounts # noqa: PLC0415
        from ..models.website_progress import WebsiteProgress # noqa: PLC0415
        address = self.address.to_dict()

        archive = self.archive.to_dict()

        helpers = self.helpers.to_dict()

        home_layout = self.home_layout.value

        members = self.members.to_dict()

        numbers = self.numbers.to_dict()

        off = self.off.to_dict()

        posts = self.posts.to_dict()

        status = self.status.value

        title = self.title

        updated_at = self.updated_at.isoformat()

        progress: dict[str, Any] | Unset = UNSET
        if not isinstance(self.progress, Unset):
            progress = self.progress.to_dict()

        turned_on_at: str | Unset = UNSET
        if not isinstance(self.turned_on_at, Unset):
            turned_on_at = self.turned_on_at.isoformat()

        turned_on_by = self.turned_on_by

        updated_by = self.updated_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "address": address,
            "archive": archive,
            "helpers": helpers,
            "homeLayout": home_layout,
            "members": members,
            "numbers": numbers,
            "off": off,
            "posts": posts,
            "status": status,
            "title": title,
            "updatedAt": updated_at,
        })
        if progress is not UNSET:
            field_dict["progress"] = progress
        if turned_on_at is not UNSET:
            field_dict["turnedOnAt"] = turned_on_at
        if turned_on_by is not UNSET:
            field_dict["turnedOnBy"] = turned_on_by
        if updated_by is not UNSET:
            field_dict["updatedBy"] = updated_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.website_address import WebsiteAddress # noqa: PLC0415
        from ..models.website_archive_facts import WebsiteArchiveFacts # noqa: PLC0415
        from ..models.website_helpers import WebsiteHelpers # noqa: PLC0415
        from ..models.website_members_facts import WebsiteMembersFacts # noqa: PLC0415
        from ..models.website_numbers import WebsiteNumbers # noqa: PLC0415
        from ..models.website_off_facts import WebsiteOffFacts # noqa: PLC0415
        from ..models.website_post_counts import WebsitePostCounts # noqa: PLC0415
        from ..models.website_progress import WebsiteProgress # noqa: PLC0415
        d = dict(src_dict)
        address = WebsiteAddress.from_dict(d.pop("address"))




        archive = WebsiteArchiveFacts.from_dict(d.pop("archive"))




        helpers = WebsiteHelpers.from_dict(d.pop("helpers"))




        home_layout = WebsiteOverviewHomeLayout(d.pop("homeLayout"))




        members = WebsiteMembersFacts.from_dict(d.pop("members"))




        numbers = WebsiteNumbers.from_dict(d.pop("numbers"))




        off = WebsiteOffFacts.from_dict(d.pop("off"))




        posts = WebsitePostCounts.from_dict(d.pop("posts"))




        status = WebsiteOverviewStatus(d.pop("status"))




        title = d.pop("title")

        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))




        _progress = d.pop("progress", UNSET)
        progress: WebsiteProgress | Unset
        if isinstance(_progress,  Unset):
            progress = UNSET
        else:
            progress = WebsiteProgress.from_dict(_progress)




        _turned_on_at = d.pop("turnedOnAt", UNSET)
        turned_on_at: datetime.datetime | Unset
        if isinstance(_turned_on_at,  Unset):
            turned_on_at = UNSET
        else:
            turned_on_at = datetime.datetime.fromisoformat(_turned_on_at)




        turned_on_by = d.pop("turnedOnBy", UNSET)

        updated_by = d.pop("updatedBy", UNSET)

        website_overview = cls(
            address=address,
            archive=archive,
            helpers=helpers,
            home_layout=home_layout,
            members=members,
            numbers=numbers,
            off=off,
            posts=posts,
            status=status,
            title=title,
            updated_at=updated_at,
            progress=progress,
            turned_on_at=turned_on_at,
            turned_on_by=turned_on_by,
            updated_by=updated_by,
        )

        return website_overview

