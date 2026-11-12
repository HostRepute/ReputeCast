from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.automation_diff_item import AutomationDiffItem
  from ..models.automation_publish_check import AutomationPublishCheck
  from ..models.automation_publish_preview_channel_steps import AutomationPublishPreviewChannelSteps





T = TypeVar("T", bound="AutomationPublishPreview")



@_attrs_define
class AutomationPublishPreview:
    """ 
        Attributes:
            can_publish (bool):
            changes (int):
            channel_steps (AutomationPublishPreviewChannelSteps):
            checks (list[AutomationPublishCheck]):
            finish_estimate (str):  Example: about 9 days.
            in_flow (int):
            items (list[AutomationDiffItem]):
            move_impact (str):
            name (str):
            requires (list[str]):
            to_version (int):
            blocker (str | Unset):
            from_version (int | Unset):
            live_published_at (datetime.datetime | Unset):
            live_published_by (str | Unset):
     """

    can_publish: bool
    changes: int
    channel_steps: AutomationPublishPreviewChannelSteps
    checks: list[AutomationPublishCheck]
    finish_estimate: str
    in_flow: int
    items: list[AutomationDiffItem]
    move_impact: str
    name: str
    requires: list[str]
    to_version: int
    blocker: str | Unset = UNSET
    from_version: int | Unset = UNSET
    live_published_at: datetime.datetime | Unset = UNSET
    live_published_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.automation_diff_item import AutomationDiffItem # noqa: PLC0415
        from ..models.automation_publish_check import AutomationPublishCheck # noqa: PLC0415
        from ..models.automation_publish_preview_channel_steps import AutomationPublishPreviewChannelSteps # noqa: PLC0415
        can_publish = self.can_publish

        changes = self.changes

        channel_steps = self.channel_steps.to_dict()

        checks = []
        for checks_item_data in self.checks:
            checks_item = checks_item_data.to_dict()
            checks.append(checks_item)



        finish_estimate = self.finish_estimate

        in_flow = self.in_flow

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        move_impact = self.move_impact

        name = self.name

        requires = self.requires



        to_version = self.to_version

        blocker = self.blocker

        from_version = self.from_version

        live_published_at: str | Unset = UNSET
        if not isinstance(self.live_published_at, Unset):
            live_published_at = self.live_published_at.isoformat()

        live_published_by = self.live_published_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canPublish": can_publish,
            "changes": changes,
            "channelSteps": channel_steps,
            "checks": checks,
            "finishEstimate": finish_estimate,
            "inFlow": in_flow,
            "items": items,
            "moveImpact": move_impact,
            "name": name,
            "requires": requires,
            "toVersion": to_version,
        })
        if blocker is not UNSET:
            field_dict["blocker"] = blocker
        if from_version is not UNSET:
            field_dict["fromVersion"] = from_version
        if live_published_at is not UNSET:
            field_dict["livePublishedAt"] = live_published_at
        if live_published_by is not UNSET:
            field_dict["livePublishedBy"] = live_published_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.automation_diff_item import AutomationDiffItem # noqa: PLC0415
        from ..models.automation_publish_check import AutomationPublishCheck # noqa: PLC0415
        from ..models.automation_publish_preview_channel_steps import AutomationPublishPreviewChannelSteps # noqa: PLC0415
        d = dict(src_dict)
        can_publish = d.pop("canPublish")

        changes = d.pop("changes")

        channel_steps = AutomationPublishPreviewChannelSteps.from_dict(d.pop("channelSteps"))




        checks = []
        _checks = d.pop("checks")
        for checks_item_data in (_checks):
            checks_item = AutomationPublishCheck.from_dict(checks_item_data)



            checks.append(checks_item)


        finish_estimate = d.pop("finishEstimate")

        in_flow = d.pop("inFlow")

        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = AutomationDiffItem.from_dict(items_item_data)



            items.append(items_item)


        move_impact = d.pop("moveImpact")

        name = d.pop("name")

        requires = cast(list[str], d.pop("requires"))


        to_version = d.pop("toVersion")

        blocker = d.pop("blocker", UNSET)

        from_version = d.pop("fromVersion", UNSET)

        _live_published_at = d.pop("livePublishedAt", UNSET)
        live_published_at: datetime.datetime | Unset
        if isinstance(_live_published_at,  Unset):
            live_published_at = UNSET
        else:
            live_published_at = datetime.datetime.fromisoformat(_live_published_at)




        live_published_by = d.pop("livePublishedBy", UNSET)

        automation_publish_preview = cls(
            can_publish=can_publish,
            changes=changes,
            channel_steps=channel_steps,
            checks=checks,
            finish_estimate=finish_estimate,
            in_flow=in_flow,
            items=items,
            move_impact=move_impact,
            name=name,
            requires=requires,
            to_version=to_version,
            blocker=blocker,
            from_version=from_version,
            live_published_at=live_published_at,
            live_published_by=live_published_by,
        )

        return automation_publish_preview

