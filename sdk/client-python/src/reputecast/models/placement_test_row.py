from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.placement_test_row_content_kind import PlacementTestRowContentKind
from ..models.placement_test_row_origin import PlacementTestRowOrigin
from ..models.placement_test_row_status import PlacementTestRowStatus
from ..models.placement_test_row_through import PlacementTestRowThrough
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.placement_counts import PlacementCounts
  from ..models.placement_mailbox_result import PlacementMailboxResult
  from ..models.placement_provider_result import PlacementProviderResult





T = TypeVar("T", bound="PlacementTestRow")



@_attrs_define
class PlacementTestRow:
    """ 
        Attributes:
            content_kind (PlacementTestRowContentKind):
            content_name (str):
            content_note (str): draft, scheduled, sent, template or the transactional template's key
            counts (PlacementCounts):
            id (str):
            mailboxes (list[PlacementMailboxResult]):
            name (str):
            origin (PlacementTestRowOrigin):
            personalization (str):
            providers (list[PlacementProviderResult]):
            started_at (datetime.datetime):
            started_by (str):
            status (PlacementTestRowStatus):
            through (PlacementTestRowThrough):
            content_at (datetime.datetime | Unset): When the campaign is scheduled or was sent
            content_id (str | Unset):
            content_key (str | Unset):
            content_version (int | Unset):
            finished_at (datetime.datetime | Unset):
            last_sent_at (datetime.datetime | Unset):
            late_until (datetime.datetime | Unset): Late copies found until then update the result
            started_by_color (str | Unset):
            started_by_id (str | Unset):
            wait_until (datetime.datetime | Unset): Copies missing then are Not arrived
     """

    content_kind: PlacementTestRowContentKind
    content_name: str
    content_note: str
    counts: PlacementCounts
    id: str
    mailboxes: list[PlacementMailboxResult]
    name: str
    origin: PlacementTestRowOrigin
    personalization: str
    providers: list[PlacementProviderResult]
    started_at: datetime.datetime
    started_by: str
    status: PlacementTestRowStatus
    through: PlacementTestRowThrough
    content_at: datetime.datetime | Unset = UNSET
    content_id: str | Unset = UNSET
    content_key: str | Unset = UNSET
    content_version: int | Unset = UNSET
    finished_at: datetime.datetime | Unset = UNSET
    last_sent_at: datetime.datetime | Unset = UNSET
    late_until: datetime.datetime | Unset = UNSET
    started_by_color: str | Unset = UNSET
    started_by_id: str | Unset = UNSET
    wait_until: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.placement_counts import PlacementCounts # noqa: PLC0415
        from ..models.placement_mailbox_result import PlacementMailboxResult # noqa: PLC0415
        from ..models.placement_provider_result import PlacementProviderResult # noqa: PLC0415
        content_kind = self.content_kind.value

        content_name = self.content_name

        content_note = self.content_note

        counts = self.counts.to_dict()

        id = self.id

        mailboxes = []
        for mailboxes_item_data in self.mailboxes:
            mailboxes_item = mailboxes_item_data.to_dict()
            mailboxes.append(mailboxes_item)



        name = self.name

        origin = self.origin.value

        personalization = self.personalization

        providers = []
        for providers_item_data in self.providers:
            providers_item = providers_item_data.to_dict()
            providers.append(providers_item)



        started_at = self.started_at.isoformat()

        started_by = self.started_by

        status = self.status.value

        through = self.through.value

        content_at: str | Unset = UNSET
        if not isinstance(self.content_at, Unset):
            content_at = self.content_at.isoformat()

        content_id = self.content_id

        content_key = self.content_key

        content_version = self.content_version

        finished_at: str | Unset = UNSET
        if not isinstance(self.finished_at, Unset):
            finished_at = self.finished_at.isoformat()

        last_sent_at: str | Unset = UNSET
        if not isinstance(self.last_sent_at, Unset):
            last_sent_at = self.last_sent_at.isoformat()

        late_until: str | Unset = UNSET
        if not isinstance(self.late_until, Unset):
            late_until = self.late_until.isoformat()

        started_by_color = self.started_by_color

        started_by_id = self.started_by_id

        wait_until: str | Unset = UNSET
        if not isinstance(self.wait_until, Unset):
            wait_until = self.wait_until.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "contentKind": content_kind,
            "contentName": content_name,
            "contentNote": content_note,
            "counts": counts,
            "id": id,
            "mailboxes": mailboxes,
            "name": name,
            "origin": origin,
            "personalization": personalization,
            "providers": providers,
            "startedAt": started_at,
            "startedBy": started_by,
            "status": status,
            "through": through,
        })
        if content_at is not UNSET:
            field_dict["contentAt"] = content_at
        if content_id is not UNSET:
            field_dict["contentId"] = content_id
        if content_key is not UNSET:
            field_dict["contentKey"] = content_key
        if content_version is not UNSET:
            field_dict["contentVersion"] = content_version
        if finished_at is not UNSET:
            field_dict["finishedAt"] = finished_at
        if last_sent_at is not UNSET:
            field_dict["lastSentAt"] = last_sent_at
        if late_until is not UNSET:
            field_dict["lateUntil"] = late_until
        if started_by_color is not UNSET:
            field_dict["startedByColor"] = started_by_color
        if started_by_id is not UNSET:
            field_dict["startedById"] = started_by_id
        if wait_until is not UNSET:
            field_dict["waitUntil"] = wait_until

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.placement_counts import PlacementCounts # noqa: PLC0415
        from ..models.placement_mailbox_result import PlacementMailboxResult # noqa: PLC0415
        from ..models.placement_provider_result import PlacementProviderResult # noqa: PLC0415
        d = dict(src_dict)
        content_kind = PlacementTestRowContentKind(d.pop("contentKind"))




        content_name = d.pop("contentName")

        content_note = d.pop("contentNote")

        counts = PlacementCounts.from_dict(d.pop("counts"))




        id = d.pop("id")

        mailboxes = []
        _mailboxes = d.pop("mailboxes")
        for mailboxes_item_data in (_mailboxes):
            mailboxes_item = PlacementMailboxResult.from_dict(mailboxes_item_data)



            mailboxes.append(mailboxes_item)


        name = d.pop("name")

        origin = PlacementTestRowOrigin(d.pop("origin"))




        personalization = d.pop("personalization")

        providers = []
        _providers = d.pop("providers")
        for providers_item_data in (_providers):
            providers_item = PlacementProviderResult.from_dict(providers_item_data)



            providers.append(providers_item)


        started_at = datetime.datetime.fromisoformat(d.pop("startedAt"))




        started_by = d.pop("startedBy")

        status = PlacementTestRowStatus(d.pop("status"))




        through = PlacementTestRowThrough(d.pop("through"))




        _content_at = d.pop("contentAt", UNSET)
        content_at: datetime.datetime | Unset
        if isinstance(_content_at,  Unset):
            content_at = UNSET
        else:
            content_at = datetime.datetime.fromisoformat(_content_at)




        content_id = d.pop("contentId", UNSET)

        content_key = d.pop("contentKey", UNSET)

        content_version = d.pop("contentVersion", UNSET)

        _finished_at = d.pop("finishedAt", UNSET)
        finished_at: datetime.datetime | Unset
        if isinstance(_finished_at,  Unset):
            finished_at = UNSET
        else:
            finished_at = datetime.datetime.fromisoformat(_finished_at)




        _last_sent_at = d.pop("lastSentAt", UNSET)
        last_sent_at: datetime.datetime | Unset
        if isinstance(_last_sent_at,  Unset):
            last_sent_at = UNSET
        else:
            last_sent_at = datetime.datetime.fromisoformat(_last_sent_at)




        _late_until = d.pop("lateUntil", UNSET)
        late_until: datetime.datetime | Unset
        if isinstance(_late_until,  Unset):
            late_until = UNSET
        else:
            late_until = datetime.datetime.fromisoformat(_late_until)




        started_by_color = d.pop("startedByColor", UNSET)

        started_by_id = d.pop("startedById", UNSET)

        _wait_until = d.pop("waitUntil", UNSET)
        wait_until: datetime.datetime | Unset
        if isinstance(_wait_until,  Unset):
            wait_until = UNSET
        else:
            wait_until = datetime.datetime.fromisoformat(_wait_until)




        placement_test_row = cls(
            content_kind=content_kind,
            content_name=content_name,
            content_note=content_note,
            counts=counts,
            id=id,
            mailboxes=mailboxes,
            name=name,
            origin=origin,
            personalization=personalization,
            providers=providers,
            started_at=started_at,
            started_by=started_by,
            status=status,
            through=through,
            content_at=content_at,
            content_id=content_id,
            content_key=content_key,
            content_version=content_version,
            finished_at=finished_at,
            last_sent_at=last_sent_at,
            late_until=late_until,
            started_by_color=started_by_color,
            started_by_id=started_by_id,
            wait_until=wait_until,
        )

        return placement_test_row

