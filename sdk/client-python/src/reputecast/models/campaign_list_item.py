from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.campaign_list_item_channel import CampaignListItemChannel
from ..models.campaign_list_item_status import CampaignListItemStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.campaign_digest_ref import CampaignDigestRef
  from ..models.readiness_summary import ReadinessSummary
  from ..models.route_share import RouteShare





T = TypeVar("T", bound="CampaignListItem")



@_attrs_define
class CampaignListItem:
    """ 
        Attributes:
            audience (str):
            blocked (bool):
            channel (CampaignListItemChannel):
            created_at (datetime.datetime):
            id (str):
            name (str):
            notes (list[str]):
            paid_only (bool):
            routes (list[RouteShare]):
            status (CampaignListItemStatus):
            subject (str):
            tags (list[str]):
            updated_at (datetime.datetime):
            updated_by (str):
            archived_at (datetime.datetime | Unset):
            clicks (float | Unset):
            cost (float | Unset):
            deleted_at (datetime.datetime | Unset):
            delivered (float | Unset):
            digest (CampaignDigestRef | Unset):
            human_opens (float | Unset):
            local (bool | Unset):
            local_time (str | Unset):
            opt_outs (int | Unset):
            origin (str | Unset):
            readiness (ReadinessSummary | Unset):
            recipients (int | Unset):
            scheduled_at (datetime.datetime | Unset):
            started_at (datetime.datetime | Unset):
            timezone (str | Unset):
     """

    audience: str
    blocked: bool
    channel: CampaignListItemChannel
    created_at: datetime.datetime
    id: str
    name: str
    notes: list[str]
    paid_only: bool
    routes: list[RouteShare]
    status: CampaignListItemStatus
    subject: str
    tags: list[str]
    updated_at: datetime.datetime
    updated_by: str
    archived_at: datetime.datetime | Unset = UNSET
    clicks: float | Unset = UNSET
    cost: float | Unset = UNSET
    deleted_at: datetime.datetime | Unset = UNSET
    delivered: float | Unset = UNSET
    digest: CampaignDigestRef | Unset = UNSET
    human_opens: float | Unset = UNSET
    local: bool | Unset = UNSET
    local_time: str | Unset = UNSET
    opt_outs: int | Unset = UNSET
    origin: str | Unset = UNSET
    readiness: ReadinessSummary | Unset = UNSET
    recipients: int | Unset = UNSET
    scheduled_at: datetime.datetime | Unset = UNSET
    started_at: datetime.datetime | Unset = UNSET
    timezone: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.campaign_digest_ref import CampaignDigestRef # noqa: PLC0415
        from ..models.readiness_summary import ReadinessSummary # noqa: PLC0415
        from ..models.route_share import RouteShare # noqa: PLC0415
        audience = self.audience

        blocked = self.blocked

        channel = self.channel.value

        created_at = self.created_at.isoformat()

        id = self.id

        name = self.name

        notes = self.notes



        paid_only = self.paid_only

        routes = []
        for routes_item_data in self.routes:
            routes_item = routes_item_data.to_dict()
            routes.append(routes_item)



        status = self.status.value

        subject = self.subject

        tags = self.tags



        updated_at = self.updated_at.isoformat()

        updated_by = self.updated_by

        archived_at: str | Unset = UNSET
        if not isinstance(self.archived_at, Unset):
            archived_at = self.archived_at.isoformat()

        clicks = self.clicks

        cost = self.cost

        deleted_at: str | Unset = UNSET
        if not isinstance(self.deleted_at, Unset):
            deleted_at = self.deleted_at.isoformat()

        delivered = self.delivered

        digest: dict[str, Any] | Unset = UNSET
        if not isinstance(self.digest, Unset):
            digest = self.digest.to_dict()

        human_opens = self.human_opens

        local = self.local

        local_time = self.local_time

        opt_outs = self.opt_outs

        origin = self.origin

        readiness: dict[str, Any] | Unset = UNSET
        if not isinstance(self.readiness, Unset):
            readiness = self.readiness.to_dict()

        recipients = self.recipients

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
            "blocked": blocked,
            "channel": channel,
            "createdAt": created_at,
            "id": id,
            "name": name,
            "notes": notes,
            "paidOnly": paid_only,
            "routes": routes,
            "status": status,
            "subject": subject,
            "tags": tags,
            "updatedAt": updated_at,
            "updatedBy": updated_by,
        })
        if archived_at is not UNSET:
            field_dict["archivedAt"] = archived_at
        if clicks is not UNSET:
            field_dict["clicks"] = clicks
        if cost is not UNSET:
            field_dict["cost"] = cost
        if deleted_at is not UNSET:
            field_dict["deletedAt"] = deleted_at
        if delivered is not UNSET:
            field_dict["delivered"] = delivered
        if digest is not UNSET:
            field_dict["digest"] = digest
        if human_opens is not UNSET:
            field_dict["humanOpens"] = human_opens
        if local is not UNSET:
            field_dict["local"] = local
        if local_time is not UNSET:
            field_dict["localTime"] = local_time
        if opt_outs is not UNSET:
            field_dict["optOuts"] = opt_outs
        if origin is not UNSET:
            field_dict["origin"] = origin
        if readiness is not UNSET:
            field_dict["readiness"] = readiness
        if recipients is not UNSET:
            field_dict["recipients"] = recipients
        if scheduled_at is not UNSET:
            field_dict["scheduledAt"] = scheduled_at
        if started_at is not UNSET:
            field_dict["startedAt"] = started_at
        if timezone is not UNSET:
            field_dict["timezone"] = timezone

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.campaign_digest_ref import CampaignDigestRef # noqa: PLC0415
        from ..models.readiness_summary import ReadinessSummary # noqa: PLC0415
        from ..models.route_share import RouteShare # noqa: PLC0415
        d = dict(src_dict)
        audience = d.pop("audience")

        blocked = d.pop("blocked")

        channel = CampaignListItemChannel(d.pop("channel"))




        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        id = d.pop("id")

        name = d.pop("name")

        notes = cast(list[str], d.pop("notes"))


        paid_only = d.pop("paidOnly")

        routes = []
        _routes = d.pop("routes")
        for routes_item_data in (_routes):
            routes_item = RouteShare.from_dict(routes_item_data)



            routes.append(routes_item)


        status = CampaignListItemStatus(d.pop("status"))




        subject = d.pop("subject")

        tags = cast(list[str], d.pop("tags"))


        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))




        updated_by = d.pop("updatedBy")

        _archived_at = d.pop("archivedAt", UNSET)
        archived_at: datetime.datetime | Unset
        if isinstance(_archived_at,  Unset):
            archived_at = UNSET
        else:
            archived_at = datetime.datetime.fromisoformat(_archived_at)




        clicks = d.pop("clicks", UNSET)

        cost = d.pop("cost", UNSET)

        _deleted_at = d.pop("deletedAt", UNSET)
        deleted_at: datetime.datetime | Unset
        if isinstance(_deleted_at,  Unset):
            deleted_at = UNSET
        else:
            deleted_at = datetime.datetime.fromisoformat(_deleted_at)




        delivered = d.pop("delivered", UNSET)

        _digest = d.pop("digest", UNSET)
        digest: CampaignDigestRef | Unset
        if isinstance(_digest,  Unset):
            digest = UNSET
        else:
            digest = CampaignDigestRef.from_dict(_digest)




        human_opens = d.pop("humanOpens", UNSET)

        local = d.pop("local", UNSET)

        local_time = d.pop("localTime", UNSET)

        opt_outs = d.pop("optOuts", UNSET)

        origin = d.pop("origin", UNSET)

        _readiness = d.pop("readiness", UNSET)
        readiness: ReadinessSummary | Unset
        if isinstance(_readiness,  Unset):
            readiness = UNSET
        else:
            readiness = ReadinessSummary.from_dict(_readiness)




        recipients = d.pop("recipients", UNSET)

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

        campaign_list_item = cls(
            audience=audience,
            blocked=blocked,
            channel=channel,
            created_at=created_at,
            id=id,
            name=name,
            notes=notes,
            paid_only=paid_only,
            routes=routes,
            status=status,
            subject=subject,
            tags=tags,
            updated_at=updated_at,
            updated_by=updated_by,
            archived_at=archived_at,
            clicks=clicks,
            cost=cost,
            deleted_at=deleted_at,
            delivered=delivered,
            digest=digest,
            human_opens=human_opens,
            local=local,
            local_time=local_time,
            opt_outs=opt_outs,
            origin=origin,
            readiness=readiness,
            recipients=recipients,
            scheduled_at=scheduled_at,
            started_at=started_at,
            timezone=timezone,
        )

        return campaign_list_item

