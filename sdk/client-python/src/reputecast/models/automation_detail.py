from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.automation_detail_status import AutomationDetailStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.automation_graph import AutomationGraph
  from ..models.automation_graph_issue import AutomationGraphIssue
  from ..models.automation_route import AutomationRoute





T = TypeVar("T", bound="AutomationDetail")



@_attrs_define
class AutomationDetail:
    """ 
        Attributes:
            changes (int): Unpublished changes, as counted by the version diff
            description (str):
            draft_version (int):
            graph (AutomationGraph):
            id (str):
            in_flow (int):
            issues (list[AutomationGraphIssue]):
            live_node_ids (list[str]):
            name (str):
            overdue (int):
            published_node_ids (list[str]):
            revision (int):
            route (AutomationRoute):
            status (AutomationDetailStatus):
            updated_at (datetime.datetime):
            live_version (int | Unset):
            pause_reason (str | Unset):
            paused_at (datetime.datetime | Unset):
            paused_by (str | Unset):
            published_at (datetime.datetime | Unset):
            published_by (str | Unset):
            template_key (str | Unset):
     """

    changes: int
    description: str
    draft_version: int
    graph: AutomationGraph
    id: str
    in_flow: int
    issues: list[AutomationGraphIssue]
    live_node_ids: list[str]
    name: str
    overdue: int
    published_node_ids: list[str]
    revision: int
    route: AutomationRoute
    status: AutomationDetailStatus
    updated_at: datetime.datetime
    live_version: int | Unset = UNSET
    pause_reason: str | Unset = UNSET
    paused_at: datetime.datetime | Unset = UNSET
    paused_by: str | Unset = UNSET
    published_at: datetime.datetime | Unset = UNSET
    published_by: str | Unset = UNSET
    template_key: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.automation_graph import AutomationGraph # noqa: PLC0415
        from ..models.automation_graph_issue import AutomationGraphIssue # noqa: PLC0415
        from ..models.automation_route import AutomationRoute # noqa: PLC0415
        changes = self.changes

        description = self.description

        draft_version = self.draft_version

        graph = self.graph.to_dict()

        id = self.id

        in_flow = self.in_flow

        issues = []
        for issues_item_data in self.issues:
            issues_item = issues_item_data.to_dict()
            issues.append(issues_item)



        live_node_ids = self.live_node_ids



        name = self.name

        overdue = self.overdue

        published_node_ids = self.published_node_ids



        revision = self.revision

        route = self.route.to_dict()

        status = self.status.value

        updated_at = self.updated_at.isoformat()

        live_version = self.live_version

        pause_reason = self.pause_reason

        paused_at: str | Unset = UNSET
        if not isinstance(self.paused_at, Unset):
            paused_at = self.paused_at.isoformat()

        paused_by = self.paused_by

        published_at: str | Unset = UNSET
        if not isinstance(self.published_at, Unset):
            published_at = self.published_at.isoformat()

        published_by = self.published_by

        template_key = self.template_key


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "changes": changes,
            "description": description,
            "draftVersion": draft_version,
            "graph": graph,
            "id": id,
            "inFlow": in_flow,
            "issues": issues,
            "liveNodeIds": live_node_ids,
            "name": name,
            "overdue": overdue,
            "publishedNodeIds": published_node_ids,
            "revision": revision,
            "route": route,
            "status": status,
            "updatedAt": updated_at,
        })
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
        if published_by is not UNSET:
            field_dict["publishedBy"] = published_by
        if template_key is not UNSET:
            field_dict["templateKey"] = template_key

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.automation_graph import AutomationGraph # noqa: PLC0415
        from ..models.automation_graph_issue import AutomationGraphIssue # noqa: PLC0415
        from ..models.automation_route import AutomationRoute # noqa: PLC0415
        d = dict(src_dict)
        changes = d.pop("changes")

        description = d.pop("description")

        draft_version = d.pop("draftVersion")

        graph = AutomationGraph.from_dict(d.pop("graph"))




        id = d.pop("id")

        in_flow = d.pop("inFlow")

        issues = []
        _issues = d.pop("issues")
        for issues_item_data in (_issues):
            issues_item = AutomationGraphIssue.from_dict(issues_item_data)



            issues.append(issues_item)


        live_node_ids = cast(list[str], d.pop("liveNodeIds"))


        name = d.pop("name")

        overdue = d.pop("overdue")

        published_node_ids = cast(list[str], d.pop("publishedNodeIds"))


        revision = d.pop("revision")

        route = AutomationRoute.from_dict(d.pop("route"))




        status = AutomationDetailStatus(d.pop("status"))




        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))




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




        published_by = d.pop("publishedBy", UNSET)

        template_key = d.pop("templateKey", UNSET)

        automation_detail = cls(
            changes=changes,
            description=description,
            draft_version=draft_version,
            graph=graph,
            id=id,
            in_flow=in_flow,
            issues=issues,
            live_node_ids=live_node_ids,
            name=name,
            overdue=overdue,
            published_node_ids=published_node_ids,
            revision=revision,
            route=route,
            status=status,
            updated_at=updated_at,
            live_version=live_version,
            pause_reason=pause_reason,
            paused_at=paused_at,
            paused_by=paused_by,
            published_at=published_at,
            published_by=published_by,
            template_key=template_key,
        )

        return automation_detail

