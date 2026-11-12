from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.automation_path_count import AutomationPathCount
  from ..models.automation_provider_sends import AutomationProviderSends





T = TypeVar("T", bound="AutomationStepStats")



@_attrs_define
class AutomationStepStats:
    """ 
        Attributes:
            bounced (int):
            clicked (int):
            complained (int):
            completed (int): Moved on to the next step
            delivered (int):
            dropped (int): Left early: filtered out, unsubscribed or removed
            entered (int):
            exited (int): Finished the flow at this step
            failed (int): Webhook calls that failed
            goal_reached (int):
            held_out (int): Contacts in the holdout group who passed the step without a message (M63)
            in_flow_now (int):
            node_id (str):
            opened (int):
            opened_human (int):
            paths (list[AutomationPathCount]):
            providers (list[AutomationProviderSends]):
            sent (int):
            skipped (int): Emails skipped by a rule, because the contact paused email on the preference center, or because
                they are in the holdout group (heldOut counts those)
            unsubscribed (int):
     """

    bounced: int
    clicked: int
    complained: int
    completed: int
    delivered: int
    dropped: int
    entered: int
    exited: int
    failed: int
    goal_reached: int
    held_out: int
    in_flow_now: int
    node_id: str
    opened: int
    opened_human: int
    paths: list[AutomationPathCount]
    providers: list[AutomationProviderSends]
    sent: int
    skipped: int
    unsubscribed: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.automation_path_count import AutomationPathCount # noqa: PLC0415
        from ..models.automation_provider_sends import AutomationProviderSends # noqa: PLC0415
        bounced = self.bounced

        clicked = self.clicked

        complained = self.complained

        completed = self.completed

        delivered = self.delivered

        dropped = self.dropped

        entered = self.entered

        exited = self.exited

        failed = self.failed

        goal_reached = self.goal_reached

        held_out = self.held_out

        in_flow_now = self.in_flow_now

        node_id = self.node_id

        opened = self.opened

        opened_human = self.opened_human

        paths = []
        for paths_item_data in self.paths:
            paths_item = paths_item_data.to_dict()
            paths.append(paths_item)



        providers = []
        for providers_item_data in self.providers:
            providers_item = providers_item_data.to_dict()
            providers.append(providers_item)



        sent = self.sent

        skipped = self.skipped

        unsubscribed = self.unsubscribed


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "bounced": bounced,
            "clicked": clicked,
            "complained": complained,
            "completed": completed,
            "delivered": delivered,
            "dropped": dropped,
            "entered": entered,
            "exited": exited,
            "failed": failed,
            "goalReached": goal_reached,
            "heldOut": held_out,
            "inFlowNow": in_flow_now,
            "nodeId": node_id,
            "opened": opened,
            "openedHuman": opened_human,
            "paths": paths,
            "providers": providers,
            "sent": sent,
            "skipped": skipped,
            "unsubscribed": unsubscribed,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.automation_path_count import AutomationPathCount # noqa: PLC0415
        from ..models.automation_provider_sends import AutomationProviderSends # noqa: PLC0415
        d = dict(src_dict)
        bounced = d.pop("bounced")

        clicked = d.pop("clicked")

        complained = d.pop("complained")

        completed = d.pop("completed")

        delivered = d.pop("delivered")

        dropped = d.pop("dropped")

        entered = d.pop("entered")

        exited = d.pop("exited")

        failed = d.pop("failed")

        goal_reached = d.pop("goalReached")

        held_out = d.pop("heldOut")

        in_flow_now = d.pop("inFlowNow")

        node_id = d.pop("nodeId")

        opened = d.pop("opened")

        opened_human = d.pop("openedHuman")

        paths = []
        _paths = d.pop("paths")
        for paths_item_data in (_paths):
            paths_item = AutomationPathCount.from_dict(paths_item_data)



            paths.append(paths_item)


        providers = []
        _providers = d.pop("providers")
        for providers_item_data in (_providers):
            providers_item = AutomationProviderSends.from_dict(providers_item_data)



            providers.append(providers_item)


        sent = d.pop("sent")

        skipped = d.pop("skipped")

        unsubscribed = d.pop("unsubscribed")

        automation_step_stats = cls(
            bounced=bounced,
            clicked=clicked,
            complained=complained,
            completed=completed,
            delivered=delivered,
            dropped=dropped,
            entered=entered,
            exited=exited,
            failed=failed,
            goal_reached=goal_reached,
            held_out=held_out,
            in_flow_now=in_flow_now,
            node_id=node_id,
            opened=opened,
            opened_human=opened_human,
            paths=paths,
            providers=providers,
            sent=sent,
            skipped=skipped,
            unsubscribed=unsubscribed,
        )

        return automation_step_stats

