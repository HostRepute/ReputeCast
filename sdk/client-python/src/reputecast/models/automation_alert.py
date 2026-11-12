from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.automation_alert_kind import AutomationAlertKind
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.automation_alert_action import AutomationAlertAction





T = TypeVar("T", bound="AutomationAlert")



@_attrs_define
class AutomationAlert:
    """ 
        Attributes:
            actions (list[AutomationAlertAction]):
            automation (str):
            automation_id (str):
            detail (str):
            failures (int):
            id (str):
            kind (AutomationAlertKind):
            since (datetime.datetime):
            title (str):
            node_id (str | Unset):
     """

    actions: list[AutomationAlertAction]
    automation: str
    automation_id: str
    detail: str
    failures: int
    id: str
    kind: AutomationAlertKind
    since: datetime.datetime
    title: str
    node_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.automation_alert_action import AutomationAlertAction # noqa: PLC0415
        actions = []
        for actions_item_data in self.actions:
            actions_item = actions_item_data.to_dict()
            actions.append(actions_item)



        automation = self.automation

        automation_id = self.automation_id

        detail = self.detail

        failures = self.failures

        id = self.id

        kind = self.kind.value

        since = self.since.isoformat()

        title = self.title

        node_id = self.node_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "actions": actions,
            "automation": automation,
            "automationId": automation_id,
            "detail": detail,
            "failures": failures,
            "id": id,
            "kind": kind,
            "since": since,
            "title": title,
        })
        if node_id is not UNSET:
            field_dict["nodeId"] = node_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.automation_alert_action import AutomationAlertAction # noqa: PLC0415
        d = dict(src_dict)
        actions = []
        _actions = d.pop("actions")
        for actions_item_data in (_actions):
            actions_item = AutomationAlertAction.from_dict(actions_item_data)



            actions.append(actions_item)


        automation = d.pop("automation")

        automation_id = d.pop("automationId")

        detail = d.pop("detail")

        failures = d.pop("failures")

        id = d.pop("id")

        kind = AutomationAlertKind(d.pop("kind"))




        since = datetime.datetime.fromisoformat(d.pop("since"))




        title = d.pop("title")

        node_id = d.pop("nodeId", UNSET)

        automation_alert = cls(
            actions=actions,
            automation=automation,
            automation_id=automation_id,
            detail=detail,
            failures=failures,
            id=id,
            kind=kind,
            since=since,
            title=title,
            node_id=node_id,
        )

        return automation_alert

