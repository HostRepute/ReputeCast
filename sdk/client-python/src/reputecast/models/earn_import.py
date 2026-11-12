from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.earn_import_mode import EarnImportMode
from ..models.earn_import_problem import EarnImportProblem
from ..models.earn_import_source import EarnImportSource
from ..models.earn_import_status import EarnImportStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.earn_import_column import EarnImportColumn
  from ..models.earn_import_conflict import EarnImportConflict
  from ..models.earn_import_decisions import EarnImportDecisions
  from ..models.earn_import_group import EarnImportGroup
  from ..models.earn_import_price import EarnImportPrice
  from ..models.earn_import_stats import EarnImportStats
  from ..models.earn_list_ref import EarnListRef





T = TypeVar("T", bound="EarnImport")



@_attrs_define
class EarnImport:
    """ 
        Attributes:
            account (str):
            columns (list[EarnImportColumn]):
            conflicts (list[EarnImportConflict]):
            created_at (datetime.datetime):
            decisions (EarnImportDecisions):
            done (int):
            file_name (str):
            file_size (int):
            id (str):
            lists (list[EarnListRef]):
            mode (EarnImportMode):
            platform (str):
            prices (list[EarnImportPrice]):
            rows (int):
            source (EarnImportSource):
            status (EarnImportStatus):
            tiers (list[EarnListRef]):
            with_customer (int):
            active (EarnImportGroup | Unset):
            created_by (str | Unset):
            ended (EarnImportGroup | Unset):
            error (str | Unset):
            finished_at (datetime.datetime | Unset):
            none (EarnImportGroup | Unset):
            paid_first (datetime.datetime | Unset):
            paid_last (datetime.datetime | Unset):
            problem (EarnImportProblem | Unset):
            started_at (datetime.datetime | Unset):
            stats (EarnImportStats | Unset):
     """

    account: str
    columns: list[EarnImportColumn]
    conflicts: list[EarnImportConflict]
    created_at: datetime.datetime
    decisions: EarnImportDecisions
    done: int
    file_name: str
    file_size: int
    id: str
    lists: list[EarnListRef]
    mode: EarnImportMode
    platform: str
    prices: list[EarnImportPrice]
    rows: int
    source: EarnImportSource
    status: EarnImportStatus
    tiers: list[EarnListRef]
    with_customer: int
    active: EarnImportGroup | Unset = UNSET
    created_by: str | Unset = UNSET
    ended: EarnImportGroup | Unset = UNSET
    error: str | Unset = UNSET
    finished_at: datetime.datetime | Unset = UNSET
    none: EarnImportGroup | Unset = UNSET
    paid_first: datetime.datetime | Unset = UNSET
    paid_last: datetime.datetime | Unset = UNSET
    problem: EarnImportProblem | Unset = UNSET
    started_at: datetime.datetime | Unset = UNSET
    stats: EarnImportStats | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.earn_import_column import EarnImportColumn # noqa: PLC0415
        from ..models.earn_import_conflict import EarnImportConflict # noqa: PLC0415
        from ..models.earn_import_decisions import EarnImportDecisions # noqa: PLC0415
        from ..models.earn_import_group import EarnImportGroup # noqa: PLC0415
        from ..models.earn_import_price import EarnImportPrice # noqa: PLC0415
        from ..models.earn_import_stats import EarnImportStats # noqa: PLC0415
        from ..models.earn_list_ref import EarnListRef # noqa: PLC0415
        account = self.account

        columns = []
        for columns_item_data in self.columns:
            columns_item = columns_item_data.to_dict()
            columns.append(columns_item)



        conflicts = []
        for conflicts_item_data in self.conflicts:
            conflicts_item = conflicts_item_data.to_dict()
            conflicts.append(conflicts_item)



        created_at = self.created_at.isoformat()

        decisions = self.decisions.to_dict()

        done = self.done

        file_name = self.file_name

        file_size = self.file_size

        id = self.id

        lists = []
        for lists_item_data in self.lists:
            lists_item = lists_item_data.to_dict()
            lists.append(lists_item)



        mode = self.mode.value

        platform = self.platform

        prices = []
        for prices_item_data in self.prices:
            prices_item = prices_item_data.to_dict()
            prices.append(prices_item)



        rows = self.rows

        source = self.source.value

        status = self.status.value

        tiers = []
        for tiers_item_data in self.tiers:
            tiers_item = tiers_item_data.to_dict()
            tiers.append(tiers_item)



        with_customer = self.with_customer

        active: dict[str, Any] | Unset = UNSET
        if not isinstance(self.active, Unset):
            active = self.active.to_dict()

        created_by = self.created_by

        ended: dict[str, Any] | Unset = UNSET
        if not isinstance(self.ended, Unset):
            ended = self.ended.to_dict()

        error = self.error

        finished_at: str | Unset = UNSET
        if not isinstance(self.finished_at, Unset):
            finished_at = self.finished_at.isoformat()

        none: dict[str, Any] | Unset = UNSET
        if not isinstance(self.none, Unset):
            none = self.none.to_dict()

        paid_first: str | Unset = UNSET
        if not isinstance(self.paid_first, Unset):
            paid_first = self.paid_first.isoformat()

        paid_last: str | Unset = UNSET
        if not isinstance(self.paid_last, Unset):
            paid_last = self.paid_last.isoformat()

        problem: str | Unset = UNSET
        if not isinstance(self.problem, Unset):
            problem = self.problem.value


        started_at: str | Unset = UNSET
        if not isinstance(self.started_at, Unset):
            started_at = self.started_at.isoformat()

        stats: dict[str, Any] | Unset = UNSET
        if not isinstance(self.stats, Unset):
            stats = self.stats.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "account": account,
            "columns": columns,
            "conflicts": conflicts,
            "createdAt": created_at,
            "decisions": decisions,
            "done": done,
            "fileName": file_name,
            "fileSize": file_size,
            "id": id,
            "lists": lists,
            "mode": mode,
            "platform": platform,
            "prices": prices,
            "rows": rows,
            "source": source,
            "status": status,
            "tiers": tiers,
            "withCustomer": with_customer,
        })
        if active is not UNSET:
            field_dict["active"] = active
        if created_by is not UNSET:
            field_dict["createdBy"] = created_by
        if ended is not UNSET:
            field_dict["ended"] = ended
        if error is not UNSET:
            field_dict["error"] = error
        if finished_at is not UNSET:
            field_dict["finishedAt"] = finished_at
        if none is not UNSET:
            field_dict["none"] = none
        if paid_first is not UNSET:
            field_dict["paidFirst"] = paid_first
        if paid_last is not UNSET:
            field_dict["paidLast"] = paid_last
        if problem is not UNSET:
            field_dict["problem"] = problem
        if started_at is not UNSET:
            field_dict["startedAt"] = started_at
        if stats is not UNSET:
            field_dict["stats"] = stats

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.earn_import_column import EarnImportColumn # noqa: PLC0415
        from ..models.earn_import_conflict import EarnImportConflict # noqa: PLC0415
        from ..models.earn_import_decisions import EarnImportDecisions # noqa: PLC0415
        from ..models.earn_import_group import EarnImportGroup # noqa: PLC0415
        from ..models.earn_import_price import EarnImportPrice # noqa: PLC0415
        from ..models.earn_import_stats import EarnImportStats # noqa: PLC0415
        from ..models.earn_list_ref import EarnListRef # noqa: PLC0415
        d = dict(src_dict)
        account = d.pop("account")

        columns = []
        _columns = d.pop("columns")
        for columns_item_data in (_columns):
            columns_item = EarnImportColumn.from_dict(columns_item_data)



            columns.append(columns_item)


        conflicts = []
        _conflicts = d.pop("conflicts")
        for conflicts_item_data in (_conflicts):
            conflicts_item = EarnImportConflict.from_dict(conflicts_item_data)



            conflicts.append(conflicts_item)


        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        decisions = EarnImportDecisions.from_dict(d.pop("decisions"))




        done = d.pop("done")

        file_name = d.pop("fileName")

        file_size = d.pop("fileSize")

        id = d.pop("id")

        lists = []
        _lists = d.pop("lists")
        for lists_item_data in (_lists):
            lists_item = EarnListRef.from_dict(lists_item_data)



            lists.append(lists_item)


        mode = EarnImportMode(d.pop("mode"))




        platform = d.pop("platform")

        prices = []
        _prices = d.pop("prices")
        for prices_item_data in (_prices):
            prices_item = EarnImportPrice.from_dict(prices_item_data)



            prices.append(prices_item)


        rows = d.pop("rows")

        source = EarnImportSource(d.pop("source"))




        status = EarnImportStatus(d.pop("status"))




        tiers = []
        _tiers = d.pop("tiers")
        for tiers_item_data in (_tiers):
            tiers_item = EarnListRef.from_dict(tiers_item_data)



            tiers.append(tiers_item)


        with_customer = d.pop("withCustomer")

        _active = d.pop("active", UNSET)
        active: EarnImportGroup | Unset
        if isinstance(_active,  Unset):
            active = UNSET
        else:
            active = EarnImportGroup.from_dict(_active)




        created_by = d.pop("createdBy", UNSET)

        _ended = d.pop("ended", UNSET)
        ended: EarnImportGroup | Unset
        if isinstance(_ended,  Unset):
            ended = UNSET
        else:
            ended = EarnImportGroup.from_dict(_ended)




        error = d.pop("error", UNSET)

        _finished_at = d.pop("finishedAt", UNSET)
        finished_at: datetime.datetime | Unset
        if isinstance(_finished_at,  Unset):
            finished_at = UNSET
        else:
            finished_at = datetime.datetime.fromisoformat(_finished_at)




        _none = d.pop("none", UNSET)
        none: EarnImportGroup | Unset
        if isinstance(_none,  Unset):
            none = UNSET
        else:
            none = EarnImportGroup.from_dict(_none)




        _paid_first = d.pop("paidFirst", UNSET)
        paid_first: datetime.datetime | Unset
        if isinstance(_paid_first,  Unset):
            paid_first = UNSET
        else:
            paid_first = datetime.datetime.fromisoformat(_paid_first)




        _paid_last = d.pop("paidLast", UNSET)
        paid_last: datetime.datetime | Unset
        if isinstance(_paid_last,  Unset):
            paid_last = UNSET
        else:
            paid_last = datetime.datetime.fromisoformat(_paid_last)




        _problem = d.pop("problem", UNSET)
        problem: EarnImportProblem | Unset
        if isinstance(_problem,  Unset):
            problem = UNSET
        else:
            problem = EarnImportProblem(_problem)




        _started_at = d.pop("startedAt", UNSET)
        started_at: datetime.datetime | Unset
        if isinstance(_started_at,  Unset):
            started_at = UNSET
        else:
            started_at = datetime.datetime.fromisoformat(_started_at)




        _stats = d.pop("stats", UNSET)
        stats: EarnImportStats | Unset
        if isinstance(_stats,  Unset):
            stats = UNSET
        else:
            stats = EarnImportStats.from_dict(_stats)




        earn_import = cls(
            account=account,
            columns=columns,
            conflicts=conflicts,
            created_at=created_at,
            decisions=decisions,
            done=done,
            file_name=file_name,
            file_size=file_size,
            id=id,
            lists=lists,
            mode=mode,
            platform=platform,
            prices=prices,
            rows=rows,
            source=source,
            status=status,
            tiers=tiers,
            with_customer=with_customer,
            active=active,
            created_by=created_by,
            ended=ended,
            error=error,
            finished_at=finished_at,
            none=none,
            paid_first=paid_first,
            paid_last=paid_last,
            problem=problem,
            started_at=started_at,
            stats=stats,
        )

        return earn_import

