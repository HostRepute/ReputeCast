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
  from ..models.page_check import PageCheck
  from ..models.page_counts import PageCounts
  from ..models.page_person import PagePerson





T = TypeVar("T", bound="PageSaved")



@_attrs_define
class PageSaved:
    """ 
        Attributes:
            changes (bool):
            checks (list[PageCheck]):
            counts (PageCounts):
            saved_at (datetime.datetime):
            version (int):
            saved_by (PagePerson | Unset):
     """

    changes: bool
    checks: list[PageCheck]
    counts: PageCounts
    saved_at: datetime.datetime
    version: int
    saved_by: PagePerson | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.page_check import PageCheck # noqa: PLC0415
        from ..models.page_counts import PageCounts # noqa: PLC0415
        from ..models.page_person import PagePerson # noqa: PLC0415
        changes = self.changes

        checks = []
        for checks_item_data in self.checks:
            checks_item = checks_item_data.to_dict()
            checks.append(checks_item)



        counts = self.counts.to_dict()

        saved_at = self.saved_at.isoformat()

        version = self.version

        saved_by: dict[str, Any] | Unset = UNSET
        if not isinstance(self.saved_by, Unset):
            saved_by = self.saved_by.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "changes": changes,
            "checks": checks,
            "counts": counts,
            "savedAt": saved_at,
            "version": version,
        })
        if saved_by is not UNSET:
            field_dict["savedBy"] = saved_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.page_check import PageCheck # noqa: PLC0415
        from ..models.page_counts import PageCounts # noqa: PLC0415
        from ..models.page_person import PagePerson # noqa: PLC0415
        d = dict(src_dict)
        changes = d.pop("changes")

        checks = []
        _checks = d.pop("checks")
        for checks_item_data in (_checks):
            checks_item = PageCheck.from_dict(checks_item_data)



            checks.append(checks_item)


        counts = PageCounts.from_dict(d.pop("counts"))




        saved_at = datetime.datetime.fromisoformat(d.pop("savedAt"))




        version = d.pop("version")

        _saved_by = d.pop("savedBy", UNSET)
        saved_by: PagePerson | Unset
        if isinstance(_saved_by,  Unset):
            saved_by = UNSET
        else:
            saved_by = PagePerson.from_dict(_saved_by)




        page_saved = cls(
            changes=changes,
            checks=checks,
            counts=counts,
            saved_at=saved_at,
            version=version,
            saved_by=saved_by,
        )

        return page_saved

