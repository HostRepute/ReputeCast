from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.visibility_catalog_field import VisibilityCatalogField
  from ..models.visibility_catalog_kind import VisibilityCatalogKind
  from ..models.visibility_catalog_options import VisibilityCatalogOptions
  from ..models.visibility_catalog_score import VisibilityCatalogScore
  from ..models.visibility_option import VisibilityOption





T = TypeVar("T", bound="VisibilityCatalog")



@_attrs_define
class VisibilityCatalog:
    """ 
        Attributes:
            bands (list[VisibilityOption]):
            fields (list[VisibilityCatalogField]):
            groups (list[str]):
            kinds (list[VisibilityCatalogKind]):
            languages (list[VisibilityOption]):
            lists (list[VisibilityOption]):
            options (VisibilityCatalogOptions):
            score (VisibilityCatalogScore):
            segments (list[VisibilityOption]):
            tags (list[VisibilityOption]):
            tiers (list[VisibilityOption]):
     """

    bands: list[VisibilityOption]
    fields: list[VisibilityCatalogField]
    groups: list[str]
    kinds: list[VisibilityCatalogKind]
    languages: list[VisibilityOption]
    lists: list[VisibilityOption]
    options: VisibilityCatalogOptions
    score: VisibilityCatalogScore
    segments: list[VisibilityOption]
    tags: list[VisibilityOption]
    tiers: list[VisibilityOption]





    def to_dict(self) -> dict[str, Any]:
        from ..models.visibility_catalog_field import VisibilityCatalogField # noqa: PLC0415
        from ..models.visibility_catalog_kind import VisibilityCatalogKind # noqa: PLC0415
        from ..models.visibility_catalog_options import VisibilityCatalogOptions # noqa: PLC0415
        from ..models.visibility_catalog_score import VisibilityCatalogScore # noqa: PLC0415
        from ..models.visibility_option import VisibilityOption # noqa: PLC0415
        bands = []
        for bands_item_data in self.bands:
            bands_item = bands_item_data.to_dict()
            bands.append(bands_item)



        fields = []
        for fields_item_data in self.fields:
            fields_item = fields_item_data.to_dict()
            fields.append(fields_item)



        groups = self.groups



        kinds = []
        for kinds_item_data in self.kinds:
            kinds_item = kinds_item_data.to_dict()
            kinds.append(kinds_item)



        languages = []
        for languages_item_data in self.languages:
            languages_item = languages_item_data.to_dict()
            languages.append(languages_item)



        lists = []
        for lists_item_data in self.lists:
            lists_item = lists_item_data.to_dict()
            lists.append(lists_item)



        options = self.options.to_dict()

        score = self.score.to_dict()

        segments = []
        for segments_item_data in self.segments:
            segments_item = segments_item_data.to_dict()
            segments.append(segments_item)



        tags = []
        for tags_item_data in self.tags:
            tags_item = tags_item_data.to_dict()
            tags.append(tags_item)



        tiers = []
        for tiers_item_data in self.tiers:
            tiers_item = tiers_item_data.to_dict()
            tiers.append(tiers_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "bands": bands,
            "fields": fields,
            "groups": groups,
            "kinds": kinds,
            "languages": languages,
            "lists": lists,
            "options": options,
            "score": score,
            "segments": segments,
            "tags": tags,
            "tiers": tiers,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.visibility_catalog_field import VisibilityCatalogField # noqa: PLC0415
        from ..models.visibility_catalog_kind import VisibilityCatalogKind # noqa: PLC0415
        from ..models.visibility_catalog_options import VisibilityCatalogOptions # noqa: PLC0415
        from ..models.visibility_catalog_score import VisibilityCatalogScore # noqa: PLC0415
        from ..models.visibility_option import VisibilityOption # noqa: PLC0415
        d = dict(src_dict)
        bands = []
        _bands = d.pop("bands")
        for bands_item_data in (_bands):
            bands_item = VisibilityOption.from_dict(bands_item_data)



            bands.append(bands_item)


        fields = []
        _fields = d.pop("fields")
        for fields_item_data in (_fields):
            fields_item = VisibilityCatalogField.from_dict(fields_item_data)



            fields.append(fields_item)


        groups = cast(list[str], d.pop("groups"))


        kinds = []
        _kinds = d.pop("kinds")
        for kinds_item_data in (_kinds):
            kinds_item = VisibilityCatalogKind.from_dict(kinds_item_data)



            kinds.append(kinds_item)


        languages = []
        _languages = d.pop("languages")
        for languages_item_data in (_languages):
            languages_item = VisibilityOption.from_dict(languages_item_data)



            languages.append(languages_item)


        lists = []
        _lists = d.pop("lists")
        for lists_item_data in (_lists):
            lists_item = VisibilityOption.from_dict(lists_item_data)



            lists.append(lists_item)


        options = VisibilityCatalogOptions.from_dict(d.pop("options"))




        score = VisibilityCatalogScore.from_dict(d.pop("score"))




        segments = []
        _segments = d.pop("segments")
        for segments_item_data in (_segments):
            segments_item = VisibilityOption.from_dict(segments_item_data)



            segments.append(segments_item)


        tags = []
        _tags = d.pop("tags")
        for tags_item_data in (_tags):
            tags_item = VisibilityOption.from_dict(tags_item_data)



            tags.append(tags_item)


        tiers = []
        _tiers = d.pop("tiers")
        for tiers_item_data in (_tiers):
            tiers_item = VisibilityOption.from_dict(tiers_item_data)



            tiers.append(tiers_item)


        visibility_catalog = cls(
            bands=bands,
            fields=fields,
            groups=groups,
            kinds=kinds,
            languages=languages,
            lists=lists,
            options=options,
            score=score,
            segments=segments,
            tags=tags,
            tiers=tiers,
        )

        return visibility_catalog

