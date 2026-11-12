from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.digest_feed_block_layout import DigestFeedBlockLayout
from ..models.digest_feed_block_order import DigestFeedBlockOrder
from ..models.digest_feed_block_overflow import DigestFeedBlockOverflow
from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="DigestFeedBlock")



@_attrs_define
class DigestFeedBlock:
    """ 
        Attributes:
            button_text (str):
            excerpt_words (int):
            fields (list[str]): image, title (linked), date, author, categories, excerpt, button
            layout (DigestFeedBlockLayout):
            max_ (int):
            order (DigestFeedBlockOrder):
            overflow (DigestFeedBlockOverflow):
            fallback (str | Unset):
            fallback_name (str | Unset):
     """

    button_text: str
    excerpt_words: int
    fields: list[str]
    layout: DigestFeedBlockLayout
    max_: int
    order: DigestFeedBlockOrder
    overflow: DigestFeedBlockOverflow
    fallback: str | Unset = UNSET
    fallback_name: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        button_text = self.button_text

        excerpt_words = self.excerpt_words

        fields = self.fields



        layout = self.layout.value

        max_ = self.max_

        order = self.order.value

        overflow = self.overflow.value

        fallback = self.fallback

        fallback_name = self.fallback_name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "buttonText": button_text,
            "excerptWords": excerpt_words,
            "fields": fields,
            "layout": layout,
            "max": max_,
            "order": order,
            "overflow": overflow,
        })
        if fallback is not UNSET:
            field_dict["fallback"] = fallback
        if fallback_name is not UNSET:
            field_dict["fallbackName"] = fallback_name

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        button_text = d.pop("buttonText")

        excerpt_words = d.pop("excerptWords")

        fields = cast(list[str], d.pop("fields"))


        layout = DigestFeedBlockLayout(d.pop("layout"))




        max_ = d.pop("max")

        order = DigestFeedBlockOrder(d.pop("order"))




        overflow = DigestFeedBlockOverflow(d.pop("overflow"))




        fallback = d.pop("fallback", UNSET)

        fallback_name = d.pop("fallbackName", UNSET)

        digest_feed_block = cls(
            button_text=button_text,
            excerpt_words=excerpt_words,
            fields=fields,
            layout=layout,
            max_=max_,
            order=order,
            overflow=overflow,
            fallback=fallback,
            fallback_name=fallback_name,
        )

        return digest_feed_block

