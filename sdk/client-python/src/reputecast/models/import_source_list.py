from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="ImportSourceList")



@_attrs_define
class ImportSourceList:
    """ 
        Attributes:
            cleaned (int): Hard bounced in the source
            complained (int):
            double_opt_in (bool):
            fields (int):
            id (str):
            name (str):
            pending (int):
            subscribed (int):
            tag_examples (list[str]):
            tags (int):
            transactional (int):
            unsubscribed (int):
            since (str | Unset):
     """

    cleaned: int
    complained: int
    double_opt_in: bool
    fields: int
    id: str
    name: str
    pending: int
    subscribed: int
    tag_examples: list[str]
    tags: int
    transactional: int
    unsubscribed: int
    since: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        cleaned = self.cleaned

        complained = self.complained

        double_opt_in = self.double_opt_in

        fields = self.fields

        id = self.id

        name = self.name

        pending = self.pending

        subscribed = self.subscribed

        tag_examples = self.tag_examples



        tags = self.tags

        transactional = self.transactional

        unsubscribed = self.unsubscribed

        since = self.since


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "cleaned": cleaned,
            "complained": complained,
            "doubleOptIn": double_opt_in,
            "fields": fields,
            "id": id,
            "name": name,
            "pending": pending,
            "subscribed": subscribed,
            "tagExamples": tag_examples,
            "tags": tags,
            "transactional": transactional,
            "unsubscribed": unsubscribed,
        })
        if since is not UNSET:
            field_dict["since"] = since

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        cleaned = d.pop("cleaned")

        complained = d.pop("complained")

        double_opt_in = d.pop("doubleOptIn")

        fields = d.pop("fields")

        id = d.pop("id")

        name = d.pop("name")

        pending = d.pop("pending")

        subscribed = d.pop("subscribed")

        tag_examples = cast(list[str], d.pop("tagExamples"))


        tags = d.pop("tags")

        transactional = d.pop("transactional")

        unsubscribed = d.pop("unsubscribed")

        since = d.pop("since", UNSET)

        import_source_list = cls(
            cleaned=cleaned,
            complained=complained,
            double_opt_in=double_opt_in,
            fields=fields,
            id=id,
            name=name,
            pending=pending,
            subscribed=subscribed,
            tag_examples=tag_examples,
            tags=tags,
            transactional=transactional,
            unsubscribed=unsubscribed,
            since=since,
        )

        return import_source_list

