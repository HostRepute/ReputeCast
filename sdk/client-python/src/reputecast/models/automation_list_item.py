from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.automation_list_item_status import AutomationListItemStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.automation_goal import AutomationGoal
  from ..models.automation_revenue import AutomationRevenue





T = TypeVar("T", bound="AutomationListItem")



@_attrs_define
class AutomationListItem:
    """ 
        Attributes:
            channels (list[str]):
            description (str):
            draft_version (int):
            id (str):
            name (str):
            status (AutomationListItemStatus):
            steps (int):
            trigger (str):
            unpublished_changes (bool):
            channel_cost (float | Unset):
            completed (float | Unset): % of entrants who finished the flow (goal or end)
            entered (int | Unset):
            goal (AutomationGoal | Unset):
            human_opens (float | Unset): Human opens, % of delivered
            in_flow (int | Unset):
            live_version (int | Unset):
            pause_reason (str | Unset):
            paused_at (datetime.datetime | Unset):
            paused_by (str | Unset):
            published_at (datetime.datetime | Unset):
            revenue (AutomationRevenue | Unset):
     """

    channels: list[str]
    description: str
    draft_version: int
    id: str
    name: str
    status: AutomationListItemStatus
    steps: int
    trigger: str
    unpublished_changes: bool
    channel_cost: float | Unset = UNSET
    completed: float | Unset = UNSET
    entered: int | Unset = UNSET
    goal: AutomationGoal | Unset = UNSET
    human_opens: float | Unset = UNSET
    in_flow: int | Unset = UNSET
    live_version: int | Unset = UNSET
    pause_reason: str | Unset = UNSET
    paused_at: datetime.datetime | Unset = UNSET
    paused_by: str | Unset = UNSET
    published_at: datetime.datetime | Unset = UNSET
    revenue: AutomationRevenue | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.automation_goal import AutomationGoal # noqa: PLC0415
        from ..models.automation_revenue import AutomationRevenue # noqa: PLC0415
        channels = self.channels



        description = self.description

        draft_version = self.draft_version

        id = self.id

        name = self.name

        status = self.status.value

        steps = self.steps

        trigger = self.trigger

        unpublished_changes = self.unpublished_changes

        channel_cost = self.channel_cost

        completed = self.completed

        entered = self.entered

        goal: dict[str, Any] | Unset = UNSET
        if not isinstance(self.goal, Unset):
            goal = self.goal.to_dict()

        human_opens = self.human_opens

        in_flow = self.in_flow

        live_version = self.live_version

        pause_reason = self.pause_reason

        paused_at: str | Unset = UNSET
        if not isinstance(self.paused_at, Unset):
            paused_at = self.paused_at.isoformat()

        paused_by = self.paused_by

        published_at: str | Unset = UNSET
        if not isinstance(self.published_at, Unset):
            published_at = self.published_at.isoformat()

        revenue: dict[str, Any] | Unset = UNSET
        if not isinstance(self.revenue, Unset):
            revenue = self.revenue.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "channels": channels,
            "description": description,
            "draftVersion": draft_version,
            "id": id,
            "name": name,
            "status": status,
            "steps": steps,
            "trigger": trigger,
            "unpublishedChanges": unpublished_changes,
        })
        if channel_cost is not UNSET:
            field_dict["channelCost"] = channel_cost
        if completed is not UNSET:
            field_dict["completed"] = completed
        if entered is not UNSET:
            field_dict["entered"] = entered
        if goal is not UNSET:
            field_dict["goal"] = goal
        if human_opens is not UNSET:
            field_dict["humanOpens"] = human_opens
        if in_flow is not UNSET:
            field_dict["inFlow"] = in_flow
        if live_version is not UNSET:
            field_dict["liveVersion"] = live_version
        if pause_reason is not UNSET:
            field_dict["pauseReason"] = pause_reason
        if paused_at is not UNSET:
            field_dict["pausedAt"] = paused_at
        if paused_by is not UNSET:
            field_dict["pausedBy"] = paused_by
        if published_at is not UNSET:
            field_dict["publishedAt"] = published_at
        if revenue is not UNSET:
            field_dict["revenue"] = revenue

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.automation_goal import AutomationGoal # noqa: PLC0415
        from ..models.automation_revenue import AutomationRevenue # noqa: PLC0415
        d = dict(src_dict)
        channels = cast(list[str], d.pop("channels"))


        description = d.pop("description")

        draft_version = d.pop("draftVersion")

        id = d.pop("id")

        name = d.pop("name")

        status = AutomationListItemStatus(d.pop("status"))




        steps = d.pop("steps")

        trigger = d.pop("trigger")

        unpublished_changes = d.pop("unpublishedChanges")

        channel_cost = d.pop("channelCost", UNSET)

        completed = d.pop("completed", UNSET)

        entered = d.pop("entered", UNSET)

        _goal = d.pop("goal", UNSET)
        goal: AutomationGoal | Unset
        if isinstance(_goal,  Unset):
            goal = UNSET
        else:
            goal = AutomationGoal.from_dict(_goal)




        human_opens = d.pop("humanOpens", UNSET)

        in_flow = d.pop("inFlow", UNSET)

        live_version = d.pop("liveVersion", UNSET)

        pause_reason = d.pop("pauseReason", UNSET)

        _paused_at = d.pop("pausedAt", UNSET)
        paused_at: datetime.datetime | Unset
        if isinstance(_paused_at,  Unset):
            paused_at = UNSET
        else:
            paused_at = datetime.datetime.fromisoformat(_paused_at)




        paused_by = d.pop("pausedBy", UNSET)

        _published_at = d.pop("publishedAt", UNSET)
        published_at: datetime.datetime | Unset
        if isinstance(_published_at,  Unset):
            published_at = UNSET
        else:
            published_at = datetime.datetime.fromisoformat(_published_at)




        _revenue = d.pop("revenue", UNSET)
        revenue: AutomationRevenue | Unset
        if isinstance(_revenue,  Unset):
            revenue = UNSET
        else:
            revenue = AutomationRevenue.from_dict(_revenue)




        automation_list_item = cls(
            channels=channels,
            description=description,
            draft_version=draft_version,
            id=id,
            name=name,
            status=status,
            steps=steps,
            trigger=trigger,
            unpublished_changes=unpublished_changes,
            channel_cost=channel_cost,
            completed=completed,
            entered=entered,
            goal=goal,
            human_opens=human_opens,
            in_flow=in_flow,
            live_version=live_version,
            pause_reason=pause_reason,
            paused_at=paused_at,
            paused_by=paused_by,
            published_at=published_at,
            revenue=revenue,
        )

        return automation_list_item

