from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.audit_diff_line import AuditDiffLine
  from ..models.audit_event_item import AuditEventItem
  from ..models.audit_restore import AuditRestore





T = TypeVar("T", bound="AuditEventDetail")



@_attrs_define
class AuditEventDetail:
    """ 
        Attributes:
            diff (list[AuditDiffLine]):
            event (AuditEventItem):
            json (Any):
            related (list[str]):
            what (str):
            workspace (str):
            after (Any | Unset):
            before (Any | Unset):
            client (str | Unset):
            ip (str | Unset):
            link (str | Unset):
            link_label (str | Unset):
            permission (str | Unset):
            restore (AuditRestore | Unset):
            user_agent (str | Unset):
     """

    diff: list[AuditDiffLine]
    event: AuditEventItem
    json: Any
    related: list[str]
    what: str
    workspace: str
    after: Any | Unset = UNSET
    before: Any | Unset = UNSET
    client: str | Unset = UNSET
    ip: str | Unset = UNSET
    link: str | Unset = UNSET
    link_label: str | Unset = UNSET
    permission: str | Unset = UNSET
    restore: AuditRestore | Unset = UNSET
    user_agent: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.audit_diff_line import AuditDiffLine # noqa: PLC0415
        from ..models.audit_event_item import AuditEventItem # noqa: PLC0415
        from ..models.audit_restore import AuditRestore # noqa: PLC0415
        diff = []
        for diff_item_data in self.diff:
            diff_item = diff_item_data.to_dict()
            diff.append(diff_item)



        event = self.event.to_dict()

        json = self.json

        related = self.related



        what = self.what

        workspace = self.workspace

        after = self.after

        before = self.before

        client = self.client

        ip = self.ip

        link = self.link

        link_label = self.link_label

        permission = self.permission

        restore: dict[str, Any] | Unset = UNSET
        if not isinstance(self.restore, Unset):
            restore = self.restore.to_dict()

        user_agent = self.user_agent


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "diff": diff,
            "event": event,
            "json": json,
            "related": related,
            "what": what,
            "workspace": workspace,
        })
        if after is not UNSET:
            field_dict["after"] = after
        if before is not UNSET:
            field_dict["before"] = before
        if client is not UNSET:
            field_dict["client"] = client
        if ip is not UNSET:
            field_dict["ip"] = ip
        if link is not UNSET:
            field_dict["link"] = link
        if link_label is not UNSET:
            field_dict["linkLabel"] = link_label
        if permission is not UNSET:
            field_dict["permission"] = permission
        if restore is not UNSET:
            field_dict["restore"] = restore
        if user_agent is not UNSET:
            field_dict["userAgent"] = user_agent

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.audit_diff_line import AuditDiffLine # noqa: PLC0415
        from ..models.audit_event_item import AuditEventItem # noqa: PLC0415
        from ..models.audit_restore import AuditRestore # noqa: PLC0415
        d = dict(src_dict)
        diff = []
        _diff = d.pop("diff")
        for diff_item_data in (_diff):
            diff_item = AuditDiffLine.from_dict(diff_item_data)



            diff.append(diff_item)


        event = AuditEventItem.from_dict(d.pop("event"))




        json = d.pop("json")

        related = cast(list[str], d.pop("related"))


        what = d.pop("what")

        workspace = d.pop("workspace")

        after = d.pop("after", UNSET)

        before = d.pop("before", UNSET)

        client = d.pop("client", UNSET)

        ip = d.pop("ip", UNSET)

        link = d.pop("link", UNSET)

        link_label = d.pop("linkLabel", UNSET)

        permission = d.pop("permission", UNSET)

        _restore = d.pop("restore", UNSET)
        restore: AuditRestore | Unset
        if isinstance(_restore,  Unset):
            restore = UNSET
        else:
            restore = AuditRestore.from_dict(_restore)




        user_agent = d.pop("userAgent", UNSET)

        audit_event_detail = cls(
            diff=diff,
            event=event,
            json=json,
            related=related,
            what=what,
            workspace=workspace,
            after=after,
            before=before,
            client=client,
            ip=ip,
            link=link,
            link_label=link_label,
            permission=permission,
            restore=restore,
            user_agent=user_agent,
        )

        return audit_event_detail

