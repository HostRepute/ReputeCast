from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="TransactionalContactOptOut")



@_attrs_define
class TransactionalContactOptOut:
    """ 
        Attributes:
            category (str):
            category_id (str):
            since (datetime.datetime):
            source (str):
            reason (str | Unset):
     """

    category: str
    category_id: str
    since: datetime.datetime
    source: str
    reason: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        category = self.category

        category_id = self.category_id

        since = self.since.isoformat()

        source = self.source

        reason = self.reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "category": category,
            "categoryId": category_id,
            "since": since,
            "source": source,
        })
        if reason is not UNSET:
            field_dict["reason"] = reason

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        category = d.pop("category")

        category_id = d.pop("categoryId")

        since = datetime.datetime.fromisoformat(d.pop("since"))




        source = d.pop("source")

        reason = d.pop("reason", UNSET)

        transactional_contact_opt_out = cls(
            category=category,
            category_id=category_id,
            since=since,
            source=source,
            reason=reason,
        )

        return transactional_contact_opt_out

