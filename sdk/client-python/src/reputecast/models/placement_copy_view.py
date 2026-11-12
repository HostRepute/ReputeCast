from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.placement_copy_view_folder import PlacementCopyViewFolder
from ..models.placement_copy_view_reason import PlacementCopyViewReason
from ..models.placement_copy_view_status import PlacementCopyViewStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.placement_event import PlacementEvent





T = TypeVar("T", bound="PlacementCopyView")



@_attrs_define
class PlacementCopyView:
    """ 
        Attributes:
            deferrals (int):
            first (bool):
            id (str):
            late (bool):
            provider_name (str):
            round_ (int):
            seed_email (str):
            seed_provider (str):
            seed_provider_name (str):
            send_at (datetime.datetime):
            status (PlacementCopyViewStatus):
            arrive_ms (int | Unset):
            dkim (str | Unset):
            dmarc (str | Unset):
            error (str | Unset): Why the provider refused the copy
            folder (PlacementCopyViewFolder | Unset):
            folder_name (str | Unset): The mailbox's own name: Junk Email, Bulk, Spamverdacht
            handed_at (datetime.datetime | Unset):
            last_event (PlacementEvent | Unset):
            late_until (datetime.datetime | Unset):
            provider_id (str | Unset):
            reason (PlacementCopyViewReason | Unset):
            reason_at (datetime.datetime | Unset):
            received_at (datetime.datetime | Unset):
            seed_id (str | Unset):
            spf (str | Unset):
            tab (str | Unset): Primary, Promotions, Focused, Other
     """

    deferrals: int
    first: bool
    id: str
    late: bool
    provider_name: str
    round_: int
    seed_email: str
    seed_provider: str
    seed_provider_name: str
    send_at: datetime.datetime
    status: PlacementCopyViewStatus
    arrive_ms: int | Unset = UNSET
    dkim: str | Unset = UNSET
    dmarc: str | Unset = UNSET
    error: str | Unset = UNSET
    folder: PlacementCopyViewFolder | Unset = UNSET
    folder_name: str | Unset = UNSET
    handed_at: datetime.datetime | Unset = UNSET
    last_event: PlacementEvent | Unset = UNSET
    late_until: datetime.datetime | Unset = UNSET
    provider_id: str | Unset = UNSET
    reason: PlacementCopyViewReason | Unset = UNSET
    reason_at: datetime.datetime | Unset = UNSET
    received_at: datetime.datetime | Unset = UNSET
    seed_id: str | Unset = UNSET
    spf: str | Unset = UNSET
    tab: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.placement_event import PlacementEvent # noqa: PLC0415
        deferrals = self.deferrals

        first = self.first

        id = self.id

        late = self.late

        provider_name = self.provider_name

        round_ = self.round_

        seed_email = self.seed_email

        seed_provider = self.seed_provider

        seed_provider_name = self.seed_provider_name

        send_at = self.send_at.isoformat()

        status = self.status.value

        arrive_ms = self.arrive_ms

        dkim = self.dkim

        dmarc = self.dmarc

        error = self.error

        folder: str | Unset = UNSET
        if not isinstance(self.folder, Unset):
            folder = self.folder.value


        folder_name = self.folder_name

        handed_at: str | Unset = UNSET
        if not isinstance(self.handed_at, Unset):
            handed_at = self.handed_at.isoformat()

        last_event: dict[str, Any] | Unset = UNSET
        if not isinstance(self.last_event, Unset):
            last_event = self.last_event.to_dict()

        late_until: str | Unset = UNSET
        if not isinstance(self.late_until, Unset):
            late_until = self.late_until.isoformat()

        provider_id = self.provider_id

        reason: str | Unset = UNSET
        if not isinstance(self.reason, Unset):
            reason = self.reason.value


        reason_at: str | Unset = UNSET
        if not isinstance(self.reason_at, Unset):
            reason_at = self.reason_at.isoformat()

        received_at: str | Unset = UNSET
        if not isinstance(self.received_at, Unset):
            received_at = self.received_at.isoformat()

        seed_id = self.seed_id

        spf = self.spf

        tab = self.tab


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "deferrals": deferrals,
            "first": first,
            "id": id,
            "late": late,
            "providerName": provider_name,
            "round": round_,
            "seedEmail": seed_email,
            "seedProvider": seed_provider,
            "seedProviderName": seed_provider_name,
            "sendAt": send_at,
            "status": status,
        })
        if arrive_ms is not UNSET:
            field_dict["arriveMs"] = arrive_ms
        if dkim is not UNSET:
            field_dict["dkim"] = dkim
        if dmarc is not UNSET:
            field_dict["dmarc"] = dmarc
        if error is not UNSET:
            field_dict["error"] = error
        if folder is not UNSET:
            field_dict["folder"] = folder
        if folder_name is not UNSET:
            field_dict["folderName"] = folder_name
        if handed_at is not UNSET:
            field_dict["handedAt"] = handed_at
        if last_event is not UNSET:
            field_dict["lastEvent"] = last_event
        if late_until is not UNSET:
            field_dict["lateUntil"] = late_until
        if provider_id is not UNSET:
            field_dict["providerId"] = provider_id
        if reason is not UNSET:
            field_dict["reason"] = reason
        if reason_at is not UNSET:
            field_dict["reasonAt"] = reason_at
        if received_at is not UNSET:
            field_dict["receivedAt"] = received_at
        if seed_id is not UNSET:
            field_dict["seedId"] = seed_id
        if spf is not UNSET:
            field_dict["spf"] = spf
        if tab is not UNSET:
            field_dict["tab"] = tab

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.placement_event import PlacementEvent # noqa: PLC0415
        d = dict(src_dict)
        deferrals = d.pop("deferrals")

        first = d.pop("first")

        id = d.pop("id")

        late = d.pop("late")

        provider_name = d.pop("providerName")

        round_ = d.pop("round")

        seed_email = d.pop("seedEmail")

        seed_provider = d.pop("seedProvider")

        seed_provider_name = d.pop("seedProviderName")

        send_at = datetime.datetime.fromisoformat(d.pop("sendAt"))




        status = PlacementCopyViewStatus(d.pop("status"))




        arrive_ms = d.pop("arriveMs", UNSET)

        dkim = d.pop("dkim", UNSET)

        dmarc = d.pop("dmarc", UNSET)

        error = d.pop("error", UNSET)

        _folder = d.pop("folder", UNSET)
        folder: PlacementCopyViewFolder | Unset
        if isinstance(_folder,  Unset):
            folder = UNSET
        else:
            folder = PlacementCopyViewFolder(_folder)




        folder_name = d.pop("folderName", UNSET)

        _handed_at = d.pop("handedAt", UNSET)
        handed_at: datetime.datetime | Unset
        if isinstance(_handed_at,  Unset):
            handed_at = UNSET
        else:
            handed_at = datetime.datetime.fromisoformat(_handed_at)




        _last_event = d.pop("lastEvent", UNSET)
        last_event: PlacementEvent | Unset
        if isinstance(_last_event,  Unset):
            last_event = UNSET
        else:
            last_event = PlacementEvent.from_dict(_last_event)




        _late_until = d.pop("lateUntil", UNSET)
        late_until: datetime.datetime | Unset
        if isinstance(_late_until,  Unset):
            late_until = UNSET
        else:
            late_until = datetime.datetime.fromisoformat(_late_until)




        provider_id = d.pop("providerId", UNSET)

        _reason = d.pop("reason", UNSET)
        reason: PlacementCopyViewReason | Unset
        if isinstance(_reason,  Unset):
            reason = UNSET
        else:
            reason = PlacementCopyViewReason(_reason)




        _reason_at = d.pop("reasonAt", UNSET)
        reason_at: datetime.datetime | Unset
        if isinstance(_reason_at,  Unset):
            reason_at = UNSET
        else:
            reason_at = datetime.datetime.fromisoformat(_reason_at)




        _received_at = d.pop("receivedAt", UNSET)
        received_at: datetime.datetime | Unset
        if isinstance(_received_at,  Unset):
            received_at = UNSET
        else:
            received_at = datetime.datetime.fromisoformat(_received_at)




        seed_id = d.pop("seedId", UNSET)

        spf = d.pop("spf", UNSET)

        tab = d.pop("tab", UNSET)

        placement_copy_view = cls(
            deferrals=deferrals,
            first=first,
            id=id,
            late=late,
            provider_name=provider_name,
            round_=round_,
            seed_email=seed_email,
            seed_provider=seed_provider,
            seed_provider_name=seed_provider_name,
            send_at=send_at,
            status=status,
            arrive_ms=arrive_ms,
            dkim=dkim,
            dmarc=dmarc,
            error=error,
            folder=folder,
            folder_name=folder_name,
            handed_at=handed_at,
            last_event=last_event,
            late_until=late_until,
            provider_id=provider_id,
            reason=reason,
            reason_at=reason_at,
            received_at=received_at,
            seed_id=seed_id,
            spf=spf,
            tab=tab,
        )

        return placement_copy_view

