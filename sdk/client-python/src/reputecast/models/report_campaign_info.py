from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.report_campaign_info_status import ReportCampaignInfoStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ReportCampaignInfo")



@_attrs_define
class ReportCampaignInfo:
    """ 
        Attributes:
            audience (str):
            final (bool):
            id (str):
            name (str):
            notes (list[str]):
            status (ReportCampaignInfoStatus):
            completed_at (datetime.datetime | Unset):
            final_at (datetime.datetime | Unset):
            origin (str | Unset):
            scheduled_at (datetime.datetime | Unset):
            started_at (datetime.datetime | Unset):
            timezone (str | Unset):
     """

    audience: str
    final: bool
    id: str
    name: str
    notes: list[str]
    status: ReportCampaignInfoStatus
    completed_at: datetime.datetime | Unset = UNSET
    final_at: datetime.datetime | Unset = UNSET
    origin: str | Unset = UNSET
    scheduled_at: datetime.datetime | Unset = UNSET
    started_at: datetime.datetime | Unset = UNSET
    timezone: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        audience = self.audience

        final = self.final

        id = self.id

        name = self.name

        notes = self.notes



        status = self.status.value

        completed_at: str | Unset = UNSET
        if not isinstance(self.completed_at, Unset):
            completed_at = self.completed_at.isoformat()

        final_at: str | Unset = UNSET
        if not isinstance(self.final_at, Unset):
            final_at = self.final_at.isoformat()

        origin = self.origin

        scheduled_at: str | Unset = UNSET
        if not isinstance(self.scheduled_at, Unset):
            scheduled_at = self.scheduled_at.isoformat()

        started_at: str | Unset = UNSET
        if not isinstance(self.started_at, Unset):
            started_at = self.started_at.isoformat()

        timezone = self.timezone


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "audience": audience,
            "final": final,
            "id": id,
            "name": name,
            "notes": notes,
            "status": status,
        })
        if completed_at is not UNSET:
            field_dict["completedAt"] = completed_at
        if final_at is not UNSET:
            field_dict["finalAt"] = final_at
        if origin is not UNSET:
            field_dict["origin"] = origin
        if scheduled_at is not UNSET:
            field_dict["scheduledAt"] = scheduled_at
        if started_at is not UNSET:
            field_dict["startedAt"] = started_at
        if timezone is not UNSET:
            field_dict["timezone"] = timezone

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        audience = d.pop("audience")

        final = d.pop("final")

        id = d.pop("id")

        name = d.pop("name")

        notes = cast(list[str], d.pop("notes"))


        status = ReportCampaignInfoStatus(d.pop("status"))




        _completed_at = d.pop("completedAt", UNSET)
        completed_at: datetime.datetime | Unset
        if isinstance(_completed_at,  Unset):
            completed_at = UNSET
        else:
            completed_at = datetime.datetime.fromisoformat(_completed_at)




        _final_at = d.pop("finalAt", UNSET)
        final_at: datetime.datetime | Unset
        if isinstance(_final_at,  Unset):
            final_at = UNSET
        else:
            final_at = datetime.datetime.fromisoformat(_final_at)




        origin = d.pop("origin", UNSET)

        _scheduled_at = d.pop("scheduledAt", UNSET)
        scheduled_at: datetime.datetime | Unset
        if isinstance(_scheduled_at,  Unset):
            scheduled_at = UNSET
        else:
            scheduled_at = datetime.datetime.fromisoformat(_scheduled_at)




        _started_at = d.pop("startedAt", UNSET)
        started_at: datetime.datetime | Unset
        if isinstance(_started_at,  Unset):
            started_at = UNSET
        else:
            started_at = datetime.datetime.fromisoformat(_started_at)




        timezone = d.pop("timezone", UNSET)

        report_campaign_info = cls(
            audience=audience,
            final=final,
            id=id,
            name=name,
            notes=notes,
            status=status,
            completed_at=completed_at,
            final_at=final_at,
            origin=origin,
            scheduled_at=scheduled_at,
            started_at=started_at,
            timezone=timezone,
        )

        return report_campaign_info

