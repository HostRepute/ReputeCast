from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.audit_event_item_result import AuditEventItemResult
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.audit_actor import AuditActor
  from ..models.audit_source import AuditSource





T = TypeVar("T", bound="AuditEventItem")



@_attrs_define
class AuditEventItem:
    """ 
        Attributes:
            action (str):
            actor (AuditActor):
            changes (bool):
            id (str):
            occurred_at (datetime.datetime):
            result (AuditEventItemResult):
            source (AuditSource):
            summary (str):
            target (str):
            target_id (str):
            target_name (str):
            target_type (str):
            reason (str | Unset):
            request_id (str | Unset):
     """

    action: str
    actor: AuditActor
    changes: bool
    id: str
    occurred_at: datetime.datetime
    result: AuditEventItemResult
    source: AuditSource
    summary: str
    target: str
    target_id: str
    target_name: str
    target_type: str
    reason: str | Unset = UNSET
    request_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.audit_actor import AuditActor # noqa: PLC0415
        from ..models.audit_source import AuditSource # noqa: PLC0415
        action = self.action

        actor = self.actor.to_dict()

        changes = self.changes

        id = self.id

        occurred_at = self.occurred_at.isoformat()

        result = self.result.value

        source = self.source.to_dict()

        summary = self.summary

        target = self.target

        target_id = self.target_id

        target_name = self.target_name

        target_type = self.target_type

        reason = self.reason

        request_id = self.request_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "action": action,
            "actor": actor,
            "changes": changes,
            "id": id,
            "occurredAt": occurred_at,
            "result": result,
            "source": source,
            "summary": summary,
            "target": target,
            "targetId": target_id,
            "targetName": target_name,
            "targetType": target_type,
        })
        if reason is not UNSET:
            field_dict["reason"] = reason
        if request_id is not UNSET:
            field_dict["requestId"] = request_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.audit_actor import AuditActor # noqa: PLC0415
        from ..models.audit_source import AuditSource # noqa: PLC0415
        d = dict(src_dict)
        action = d.pop("action")

        actor = AuditActor.from_dict(d.pop("actor"))




        changes = d.pop("changes")

        id = d.pop("id")

        occurred_at = datetime.datetime.fromisoformat(d.pop("occurredAt"))




        result = AuditEventItemResult(d.pop("result"))




        source = AuditSource.from_dict(d.pop("source"))




        summary = d.pop("summary")

        target = d.pop("target")

        target_id = d.pop("targetId")

        target_name = d.pop("targetName")

        target_type = d.pop("targetType")

        reason = d.pop("reason", UNSET)

        request_id = d.pop("requestId", UNSET)

        audit_event_item = cls(
            action=action,
            actor=actor,
            changes=changes,
            id=id,
            occurred_at=occurred_at,
            result=result,
            source=source,
            summary=summary,
            target=target,
            target_id=target_id,
            target_name=target_name,
            target_type=target_type,
            reason=reason,
            request_id=request_id,
        )

        return audit_event_item

