from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.automation_contact_run_status import AutomationContactRunStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="AutomationContactRun")



@_attrs_define
class AutomationContactRun:
    """ 
        Attributes:
            automation_id (str):
            entered_at (datetime.datetime):
            name (str):
            node_id (str):
            run_id (str):
            status (AutomationContactRunStatus):
            step (str):
            step_since (datetime.datetime):
            exit_reason (str | Unset):
            finished_at (datetime.datetime | Unset):
     """

    automation_id: str
    entered_at: datetime.datetime
    name: str
    node_id: str
    run_id: str
    status: AutomationContactRunStatus
    step: str
    step_since: datetime.datetime
    exit_reason: str | Unset = UNSET
    finished_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        automation_id = self.automation_id

        entered_at = self.entered_at.isoformat()

        name = self.name

        node_id = self.node_id

        run_id = self.run_id

        status = self.status.value

        step = self.step

        step_since = self.step_since.isoformat()

        exit_reason = self.exit_reason

        finished_at: str | Unset = UNSET
        if not isinstance(self.finished_at, Unset):
            finished_at = self.finished_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "automationId": automation_id,
            "enteredAt": entered_at,
            "name": name,
            "nodeId": node_id,
            "runId": run_id,
            "status": status,
            "step": step,
            "stepSince": step_since,
        })
        if exit_reason is not UNSET:
            field_dict["exitReason"] = exit_reason
        if finished_at is not UNSET:
            field_dict["finishedAt"] = finished_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        automation_id = d.pop("automationId")

        entered_at = datetime.datetime.fromisoformat(d.pop("enteredAt"))




        name = d.pop("name")

        node_id = d.pop("nodeId")

        run_id = d.pop("runId")

        status = AutomationContactRunStatus(d.pop("status"))




        step = d.pop("step")

        step_since = datetime.datetime.fromisoformat(d.pop("stepSince"))




        exit_reason = d.pop("exitReason", UNSET)

        _finished_at = d.pop("finishedAt", UNSET)
        finished_at: datetime.datetime | Unset
        if isinstance(_finished_at,  Unset):
            finished_at = UNSET
        else:
            finished_at = datetime.datetime.fromisoformat(_finished_at)




        automation_contact_run = cls(
            automation_id=automation_id,
            entered_at=entered_at,
            name=name,
            node_id=node_id,
            run_id=run_id,
            status=status,
            step=step,
            step_since=step_since,
            exit_reason=exit_reason,
            finished_at=finished_at,
        )

        return automation_contact_run

