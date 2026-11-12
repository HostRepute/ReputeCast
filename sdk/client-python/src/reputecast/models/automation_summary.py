from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.automation_goal_count import AutomationGoalCount
  from ..models.automation_provider_sends import AutomationProviderSends





T = TypeVar("T", bound="AutomationSummary")



@_attrs_define
class AutomationSummary:
    """ 
        Attributes:
            days (int):
            goals (int):
            goals_by (list[AutomationGoalCount]):
            in_flow (int):
            live_count (int):
            sent (int):
            sent_by (list[AutomationProviderSends]):
            clicks (float | Unset):
            human_opens (float | Unset):
     """

    days: int
    goals: int
    goals_by: list[AutomationGoalCount]
    in_flow: int
    live_count: int
    sent: int
    sent_by: list[AutomationProviderSends]
    clicks: float | Unset = UNSET
    human_opens: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.automation_goal_count import AutomationGoalCount # noqa: PLC0415
        from ..models.automation_provider_sends import AutomationProviderSends # noqa: PLC0415
        days = self.days

        goals = self.goals

        goals_by = []
        for goals_by_item_data in self.goals_by:
            goals_by_item = goals_by_item_data.to_dict()
            goals_by.append(goals_by_item)



        in_flow = self.in_flow

        live_count = self.live_count

        sent = self.sent

        sent_by = []
        for sent_by_item_data in self.sent_by:
            sent_by_item = sent_by_item_data.to_dict()
            sent_by.append(sent_by_item)



        clicks = self.clicks

        human_opens = self.human_opens


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "days": days,
            "goals": goals,
            "goalsBy": goals_by,
            "inFlow": in_flow,
            "liveCount": live_count,
            "sent": sent,
            "sentBy": sent_by,
        })
        if clicks is not UNSET:
            field_dict["clicks"] = clicks
        if human_opens is not UNSET:
            field_dict["humanOpens"] = human_opens

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.automation_goal_count import AutomationGoalCount # noqa: PLC0415
        from ..models.automation_provider_sends import AutomationProviderSends # noqa: PLC0415
        d = dict(src_dict)
        days = d.pop("days")

        goals = d.pop("goals")

        goals_by = []
        _goals_by = d.pop("goalsBy")
        for goals_by_item_data in (_goals_by):
            goals_by_item = AutomationGoalCount.from_dict(goals_by_item_data)



            goals_by.append(goals_by_item)


        in_flow = d.pop("inFlow")

        live_count = d.pop("liveCount")

        sent = d.pop("sent")

        sent_by = []
        _sent_by = d.pop("sentBy")
        for sent_by_item_data in (_sent_by):
            sent_by_item = AutomationProviderSends.from_dict(sent_by_item_data)



            sent_by.append(sent_by_item)


        clicks = d.pop("clicks", UNSET)

        human_opens = d.pop("humanOpens", UNSET)

        automation_summary = cls(
            days=days,
            goals=goals,
            goals_by=goals_by,
            in_flow=in_flow,
            live_count=live_count,
            sent=sent,
            sent_by=sent_by,
            clicks=clicks,
            human_opens=human_opens,
        )

        return automation_summary

