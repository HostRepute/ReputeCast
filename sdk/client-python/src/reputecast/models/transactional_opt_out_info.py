from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transactional_opt_out_info_source import TransactionalOptOutInfoSource
from typing import cast
import datetime






T = TypeVar("T", bound="TransactionalOptOutInfo")



@_attrs_define
class TransactionalOptOutInfo:
    """ 
        Attributes:
            category (str):
            since (datetime.datetime):
            source (TransactionalOptOutInfoSource):
     """

    category: str
    since: datetime.datetime
    source: TransactionalOptOutInfoSource





    def to_dict(self) -> dict[str, Any]:
        category = self.category

        since = self.since.isoformat()

        source = self.source.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "category": category,
            "since": since,
            "source": source,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        category = d.pop("category")

        since = datetime.datetime.fromisoformat(d.pop("since"))




        source = TransactionalOptOutInfoSource(d.pop("source"))




        transactional_opt_out_info = cls(
            category=category,
            since=since,
            source=source,
        )

        return transactional_opt_out_info

