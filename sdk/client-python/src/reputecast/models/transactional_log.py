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
  from ..models.transactional_log_counts import TransactionalLogCounts
  from ..models.transactional_log_filters import TransactionalLogFilters
  from ..models.transactional_log_item import TransactionalLogItem





T = TypeVar("T", bound="TransactionalLog")



@_attrs_define
class TransactionalLog:
    """ 
        Attributes:
            can_export (bool):
            content_days (int):
            counts (TransactionalLogCounts):
            filters (TransactionalLogFilters):
            items (list[TransactionalLogItem]):
            metadata_days (int):
            since (datetime.datetime):
            until (datetime.datetime):
            newest (str | Unset):
            next_ (str | Unset):
     """

    can_export: bool
    content_days: int
    counts: TransactionalLogCounts
    filters: TransactionalLogFilters
    items: list[TransactionalLogItem]
    metadata_days: int
    since: datetime.datetime
    until: datetime.datetime
    newest: str | Unset = UNSET
    next_: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_log_counts import TransactionalLogCounts # noqa: PLC0415
        from ..models.transactional_log_filters import TransactionalLogFilters # noqa: PLC0415
        from ..models.transactional_log_item import TransactionalLogItem # noqa: PLC0415
        can_export = self.can_export

        content_days = self.content_days

        counts = self.counts.to_dict()

        filters = self.filters.to_dict()

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        metadata_days = self.metadata_days

        since = self.since.isoformat()

        until = self.until.isoformat()

        newest = self.newest

        next_ = self.next_


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canExport": can_export,
            "contentDays": content_days,
            "counts": counts,
            "filters": filters,
            "items": items,
            "metadataDays": metadata_days,
            "since": since,
            "until": until,
        })
        if newest is not UNSET:
            field_dict["newest"] = newest
        if next_ is not UNSET:
            field_dict["next"] = next_

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_log_counts import TransactionalLogCounts # noqa: PLC0415
        from ..models.transactional_log_filters import TransactionalLogFilters # noqa: PLC0415
        from ..models.transactional_log_item import TransactionalLogItem # noqa: PLC0415
        d = dict(src_dict)
        can_export = d.pop("canExport")

        content_days = d.pop("contentDays")

        counts = TransactionalLogCounts.from_dict(d.pop("counts"))




        filters = TransactionalLogFilters.from_dict(d.pop("filters"))




        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = TransactionalLogItem.from_dict(items_item_data)



            items.append(items_item)


        metadata_days = d.pop("metadataDays")

        since = datetime.datetime.fromisoformat(d.pop("since"))




        until = datetime.datetime.fromisoformat(d.pop("until"))




        newest = d.pop("newest", UNSET)

        next_ = d.pop("next", UNSET)

        transactional_log = cls(
            can_export=can_export,
            content_days=content_days,
            counts=counts,
            filters=filters,
            items=items,
            metadata_days=metadata_days,
            since=since,
            until=until,
            newest=newest,
            next_=next_,
        )

        return transactional_log

