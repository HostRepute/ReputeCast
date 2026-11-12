from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.earn_webhook_status import EarnWebhookStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.earn_event_count import EarnEventCount
  from ..models.earn_gap_run import EarnGapRun
  from ..models.earn_reach import EarnReach





T = TypeVar("T", bound="EarnWebhook")



@_attrs_define
class EarnWebhook:
    """ 
        Attributes:
            by_type (list[EarnEventCount]):
            events (list[str]):
            events24h (int):
            gap_runs (list[EarnGapRun]):
            manual (bool):
            reach (EarnReach):
            registered (bool):
            rejected24h (int):
            secret_stored (bool):
            status (EarnWebhookStatus):
            url (str):
            first_event_at (datetime.datetime | Unset):
            first_event_type (str | Unset):
            gap_checked_at (datetime.datetime | Unset):
            id (str | Unset):
            last_event_at (datetime.datetime | Unset):
            last_event_type (str | Unset):
            missing_at (datetime.datetime | Unset):
            next_gap_at (datetime.datetime | Unset):
            registered_at (datetime.datetime | Unset):
            silent_hours (int | Unset):
     """

    by_type: list[EarnEventCount]
    events: list[str]
    events24h: int
    gap_runs: list[EarnGapRun]
    manual: bool
    reach: EarnReach
    registered: bool
    rejected24h: int
    secret_stored: bool
    status: EarnWebhookStatus
    url: str
    first_event_at: datetime.datetime | Unset = UNSET
    first_event_type: str | Unset = UNSET
    gap_checked_at: datetime.datetime | Unset = UNSET
    id: str | Unset = UNSET
    last_event_at: datetime.datetime | Unset = UNSET
    last_event_type: str | Unset = UNSET
    missing_at: datetime.datetime | Unset = UNSET
    next_gap_at: datetime.datetime | Unset = UNSET
    registered_at: datetime.datetime | Unset = UNSET
    silent_hours: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.earn_event_count import EarnEventCount # noqa: PLC0415
        from ..models.earn_gap_run import EarnGapRun # noqa: PLC0415
        from ..models.earn_reach import EarnReach # noqa: PLC0415
        by_type = []
        for by_type_item_data in self.by_type:
            by_type_item = by_type_item_data.to_dict()
            by_type.append(by_type_item)



        events = self.events



        events24h = self.events24h

        gap_runs = []
        for gap_runs_item_data in self.gap_runs:
            gap_runs_item = gap_runs_item_data.to_dict()
            gap_runs.append(gap_runs_item)



        manual = self.manual

        reach = self.reach.to_dict()

        registered = self.registered

        rejected24h = self.rejected24h

        secret_stored = self.secret_stored

        status = self.status.value

        url = self.url

        first_event_at: str | Unset = UNSET
        if not isinstance(self.first_event_at, Unset):
            first_event_at = self.first_event_at.isoformat()

        first_event_type = self.first_event_type

        gap_checked_at: str | Unset = UNSET
        if not isinstance(self.gap_checked_at, Unset):
            gap_checked_at = self.gap_checked_at.isoformat()

        id = self.id

        last_event_at: str | Unset = UNSET
        if not isinstance(self.last_event_at, Unset):
            last_event_at = self.last_event_at.isoformat()

        last_event_type = self.last_event_type

        missing_at: str | Unset = UNSET
        if not isinstance(self.missing_at, Unset):
            missing_at = self.missing_at.isoformat()

        next_gap_at: str | Unset = UNSET
        if not isinstance(self.next_gap_at, Unset):
            next_gap_at = self.next_gap_at.isoformat()

        registered_at: str | Unset = UNSET
        if not isinstance(self.registered_at, Unset):
            registered_at = self.registered_at.isoformat()

        silent_hours = self.silent_hours


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "byType": by_type,
            "events": events,
            "events24h": events24h,
            "gapRuns": gap_runs,
            "manual": manual,
            "reach": reach,
            "registered": registered,
            "rejected24h": rejected24h,
            "secretStored": secret_stored,
            "status": status,
            "url": url,
        })
        if first_event_at is not UNSET:
            field_dict["firstEventAt"] = first_event_at
        if first_event_type is not UNSET:
            field_dict["firstEventType"] = first_event_type
        if gap_checked_at is not UNSET:
            field_dict["gapCheckedAt"] = gap_checked_at
        if id is not UNSET:
            field_dict["id"] = id
        if last_event_at is not UNSET:
            field_dict["lastEventAt"] = last_event_at
        if last_event_type is not UNSET:
            field_dict["lastEventType"] = last_event_type
        if missing_at is not UNSET:
            field_dict["missingAt"] = missing_at
        if next_gap_at is not UNSET:
            field_dict["nextGapAt"] = next_gap_at
        if registered_at is not UNSET:
            field_dict["registeredAt"] = registered_at
        if silent_hours is not UNSET:
            field_dict["silentHours"] = silent_hours

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.earn_event_count import EarnEventCount # noqa: PLC0415
        from ..models.earn_gap_run import EarnGapRun # noqa: PLC0415
        from ..models.earn_reach import EarnReach # noqa: PLC0415
        d = dict(src_dict)
        by_type = []
        _by_type = d.pop("byType")
        for by_type_item_data in (_by_type):
            by_type_item = EarnEventCount.from_dict(by_type_item_data)



            by_type.append(by_type_item)


        events = cast(list[str], d.pop("events"))


        events24h = d.pop("events24h")

        gap_runs = []
        _gap_runs = d.pop("gapRuns")
        for gap_runs_item_data in (_gap_runs):
            gap_runs_item = EarnGapRun.from_dict(gap_runs_item_data)



            gap_runs.append(gap_runs_item)


        manual = d.pop("manual")

        reach = EarnReach.from_dict(d.pop("reach"))




        registered = d.pop("registered")

        rejected24h = d.pop("rejected24h")

        secret_stored = d.pop("secretStored")

        status = EarnWebhookStatus(d.pop("status"))




        url = d.pop("url")

        _first_event_at = d.pop("firstEventAt", UNSET)
        first_event_at: datetime.datetime | Unset
        if isinstance(_first_event_at,  Unset):
            first_event_at = UNSET
        else:
            first_event_at = datetime.datetime.fromisoformat(_first_event_at)




        first_event_type = d.pop("firstEventType", UNSET)

        _gap_checked_at = d.pop("gapCheckedAt", UNSET)
        gap_checked_at: datetime.datetime | Unset
        if isinstance(_gap_checked_at,  Unset):
            gap_checked_at = UNSET
        else:
            gap_checked_at = datetime.datetime.fromisoformat(_gap_checked_at)




        id = d.pop("id", UNSET)

        _last_event_at = d.pop("lastEventAt", UNSET)
        last_event_at: datetime.datetime | Unset
        if isinstance(_last_event_at,  Unset):
            last_event_at = UNSET
        else:
            last_event_at = datetime.datetime.fromisoformat(_last_event_at)




        last_event_type = d.pop("lastEventType", UNSET)

        _missing_at = d.pop("missingAt", UNSET)
        missing_at: datetime.datetime | Unset
        if isinstance(_missing_at,  Unset):
            missing_at = UNSET
        else:
            missing_at = datetime.datetime.fromisoformat(_missing_at)




        _next_gap_at = d.pop("nextGapAt", UNSET)
        next_gap_at: datetime.datetime | Unset
        if isinstance(_next_gap_at,  Unset):
            next_gap_at = UNSET
        else:
            next_gap_at = datetime.datetime.fromisoformat(_next_gap_at)




        _registered_at = d.pop("registeredAt", UNSET)
        registered_at: datetime.datetime | Unset
        if isinstance(_registered_at,  Unset):
            registered_at = UNSET
        else:
            registered_at = datetime.datetime.fromisoformat(_registered_at)




        silent_hours = d.pop("silentHours", UNSET)

        earn_webhook = cls(
            by_type=by_type,
            events=events,
            events24h=events24h,
            gap_runs=gap_runs,
            manual=manual,
            reach=reach,
            registered=registered,
            rejected24h=rejected24h,
            secret_stored=secret_stored,
            status=status,
            url=url,
            first_event_at=first_event_at,
            first_event_type=first_event_type,
            gap_checked_at=gap_checked_at,
            id=id,
            last_event_at=last_event_at,
            last_event_type=last_event_type,
            missing_at=missing_at,
            next_gap_at=next_gap_at,
            registered_at=registered_at,
            silent_hours=silent_hours,
        )

        return earn_webhook

