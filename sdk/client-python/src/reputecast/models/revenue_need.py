from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.revenue_need_kind import RevenueNeedKind
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="RevenueNeed")



@_attrs_define
class RevenueNeed:
    """ 
        Attributes:
            kind (RevenueNeedKind):
            count (int | Unset):
            due (datetime.datetime | Unset):
            ends_at (datetime.datetime | Unset):
            issue (int | Unset):
            limit (int | Unset):
            list_ (str | Unset):
            name (str | Unset):
            overdue (bool | Unset):
            sponsor (str | Unset):
            target_id (str | Unset):
            template (str | Unset):
            used (int | Unset):
     """

    kind: RevenueNeedKind
    count: int | Unset = UNSET
    due: datetime.datetime | Unset = UNSET
    ends_at: datetime.datetime | Unset = UNSET
    issue: int | Unset = UNSET
    limit: int | Unset = UNSET
    list_: str | Unset = UNSET
    name: str | Unset = UNSET
    overdue: bool | Unset = UNSET
    sponsor: str | Unset = UNSET
    target_id: str | Unset = UNSET
    template: str | Unset = UNSET
    used: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind.value

        count = self.count

        due: str | Unset = UNSET
        if not isinstance(self.due, Unset):
            due = self.due.isoformat()

        ends_at: str | Unset = UNSET
        if not isinstance(self.ends_at, Unset):
            ends_at = self.ends_at.isoformat()

        issue = self.issue

        limit = self.limit

        list_ = self.list_

        name = self.name

        overdue = self.overdue

        sponsor = self.sponsor

        target_id = self.target_id

        template = self.template

        used = self.used


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
        })
        if count is not UNSET:
            field_dict["count"] = count
        if due is not UNSET:
            field_dict["due"] = due
        if ends_at is not UNSET:
            field_dict["endsAt"] = ends_at
        if issue is not UNSET:
            field_dict["issue"] = issue
        if limit is not UNSET:
            field_dict["limit"] = limit
        if list_ is not UNSET:
            field_dict["list"] = list_
        if name is not UNSET:
            field_dict["name"] = name
        if overdue is not UNSET:
            field_dict["overdue"] = overdue
        if sponsor is not UNSET:
            field_dict["sponsor"] = sponsor
        if target_id is not UNSET:
            field_dict["targetId"] = target_id
        if template is not UNSET:
            field_dict["template"] = template
        if used is not UNSET:
            field_dict["used"] = used

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = RevenueNeedKind(d.pop("kind"))




        count = d.pop("count", UNSET)

        _due = d.pop("due", UNSET)
        due: datetime.datetime | Unset
        if isinstance(_due,  Unset):
            due = UNSET
        else:
            due = datetime.datetime.fromisoformat(_due)




        _ends_at = d.pop("endsAt", UNSET)
        ends_at: datetime.datetime | Unset
        if isinstance(_ends_at,  Unset):
            ends_at = UNSET
        else:
            ends_at = datetime.datetime.fromisoformat(_ends_at)




        issue = d.pop("issue", UNSET)

        limit = d.pop("limit", UNSET)

        list_ = d.pop("list", UNSET)

        name = d.pop("name", UNSET)

        overdue = d.pop("overdue", UNSET)

        sponsor = d.pop("sponsor", UNSET)

        target_id = d.pop("targetId", UNSET)

        template = d.pop("template", UNSET)

        used = d.pop("used", UNSET)

        revenue_need = cls(
            kind=kind,
            count=count,
            due=due,
            ends_at=ends_at,
            issue=issue,
            limit=limit,
            list_=list_,
            name=name,
            overdue=overdue,
            sponsor=sponsor,
            target_id=target_id,
            template=template,
            used=used,
        )

        return revenue_need

