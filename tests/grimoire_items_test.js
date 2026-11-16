'use strict';
const test=require('node:test'),assert=require('node:assert/strict');
const fs=require('node:fs'),path=require('node:path');
const boot=require('./helpers/grimoire-vm');
function fixture(t){
 const vm=boot(t);
 vm.run(`
  ITEMS={
   {key='bags:1',id=1,name='Azure Cloth Hat',classID=4,subclassID=1,subtype='Cloth',slot='INVTYPE_HEAD',quality=2,level=10},
   {key='bags:2',id=2,name='Brown Leather Hat',classID=4,subclassID=2,subtype='Leather',slot='INVTYPE_HEAD',quality=1,level=20},
   {key='bags:3',id=3,name='Great Axe',classID=2,subclassID=1,subtype='Two-handed axes',slot='INVTYPE_2HWEAPON',quality=3,level=30},
   {key='bags:4',id=4,name='Uncached Thing'},
   {key='bags:5',id=5,name='Unclassified Armor',classID=4},
   {key='bags:6',id=6,name='Necklace',classID=4,subclassID=0,slot='INVTYPE_NECK'},
  }
  for _,item in ipairs(ITEMS) do item.link='item:'..item.id;item.count=1;item.location='Bags';item.kind='';item.subtype=item.subtype or '';item.slot=item.slot or '' end
  AzerMind.OwnedItems=function() return ITEMS end
  AzerMind.ShowItemPicker();P=AzerMindItemPicker
 `);
 return vm;
}
test('item tree separates family and subtype IDs, expands without losing filters and retains unknowns',t=>{
 const vm=fixture(t);
 vm.run(`
  P.SelectCategory('class:4');assert(#P.visible==4 and P.expanded['class:4'],'armor parent incomplete')
  P.SelectCategory('sub:4:1');assert(#P.visible==1 and P.visible[1].id==1,'cloth includes another subclass')
  P.selected['bags:1']=true;P.ToggleCategory('class:4')
  assert(P.category=='sub:4:1' and P.selected['bags:1'],'collapse changed filter or selection')
  P.SelectCategory('sub:2:1');assert(#P.visible==1 and P.visible[1].id==3,'subclass collided across classes')
  P.SelectCategory('other');assert(#P.visible==1 and P.visible[1].id==4,'unknown class disappeared')
  P.SelectCategory('unknown:4');assert(#P.visible==1 and P.visible[1].id==5,'unknown subtype disappeared')
  P.SelectCategory('accessories');assert(#P.visible==1 and P.visible[1].id==6,'slot shortcut lost')
  P.SelectCategory('all');assert(#P.visible==6,'items became unreachable')
 `);
 assert.equal(vm.value('P.subtypeSelector'),'nil');
});

test('item window reserves separate tree, result, paging and selection regions and fits smaller screens',t=>{
 const vm=fixture(t);
 vm.run(fs.readFileSync(path.join(__dirname,'helpers/grimoire-geometry.lua'),'utf8'));
 vm.run(`
  assert(P.grimoireFrame.name=='frame-window','item window lacks outer frame')
  local tree,list=GR_Bounds(P.categoryScroll),GR_Bounds(P.scroll)
  assert(tree.right+30<=list.left,'category scrollbar crowds results')
  local previous,next=GR_Bounds(P.previous),GR_Bounds(P.next)
  assert(previous.top<list.bottom and next.top<list.bottom,'paging overlaps results')
  local insert=GR_Bounds(P.insert)
  for _,button in ipairs(P.children) do
   if button.text=='Select page' or button.text=='Select all results' or button.text=='Clear selection' then
    local b=GR_Bounds(button);assert(b.top<previous.bottom and b.bottom>insert.top,'selection actions mixed with search or footer')
   end
  end
  for _,size in ipairs({{900,600},{1280,720},{1920,1080}}) do
   UIParent:SetSize(size[1],size[2])
   for _,fn in ipairs(UIParent.hooks.OnSizeChanged or {}) do fn(UIParent,size[1],size[2]) end
   assert(P:GetWidth()*P:GetScale()<=size[1]-32,'window exceeds screen width')
   assert(P:GetHeight()*P:GetScale()<=size[2]-32,'window exceeds screen height')
  end
 `);
});

test('word, ID and pasted-link searches stay scoped and can broaden without changing item payloads',t=>{
 const vm=fixture(t);
 vm.run(`
  function SEARCH(text) P.search:SetText(text);P.search.scripts.OnEnterPressed() end
  P.selected['bags:1']=true; SEARCH('hat AZURE')
  assert(#P.visible==1 and P.visible[1].id==1,'word search did not match independently')
  P.SelectCategory('class:2');assert(#P.visible==0 and P.searchAll:IsShown(),'scope was silently ignored')
  P.searchAll.scripts.OnClick();assert(#P.visible==1 and P.category=='all','Search all failed')
  SEARCH('3');assert(#P.visible==1 and P.visible[1].id==3,'numeric ID search failed')
  SEARCH('|cff00ff00|Hitem:3:0:0:0:0:0:99|h[Different label]|h|r')
  assert(#P.visible==1 and P.visible[1].link=='item:3','pasted search replaced item payload')
  P.ClearFilter('all');assert(#P.visible==6 and P.selected['bags:1'],'reset cleared selection')
 `);
});

test('combined filters expose counts and incomplete metadata, refreshing only once for queued updates',t=>{
 const vm=fixture(t);
 vm.run(`
  P.minInput:SetText('15');P.minInput.scripts.OnTextChanged(P.minInput)
  P.maxInput:SetText('25');P.maxInput.scripts.OnTextChanged(P.maxInput)
  assert(#P.visible==1 and P.visible[1].id==2,'required level range wrong')
  assert(P.categoryCounts['class:4']==1 and not P.categoryCounts['sub:4:1'],'tree counts ignored filters')
  P.minInput:SetText('30');P.minInput.scripts.OnTextChanged(P.minInput)
  assert(#P.visible==0 and P.empty:GetText():find('Min level',1,true),'invalid range not explained')
  P.ClearFilter('level');P.rarity=4;P.Refresh()
  assert(#P.visible==0 and P.unknownMetadata>0,'missing metadata claimed as no results')
  local passes=P.filterPasses;P.ToggleCategory('class:4')
  for _,fn in ipairs(P.scroll.hooks.OnVerticalScroll) do fn() end
  assert(P.filterPasses==passes,'scroll/expansion rescanned all items')
  ITEMS[4].quality=4
  STUB.FireEvent('GET_ITEM_INFO_RECEIVED');STUB.FireEvent('GET_ITEM_INFO_RECEIVED');STUB.RunTimers()
  assert(#P.visible==1 and P.visible[1].id==4,'fresh metadata did not invalidate filter cache')
  assert(P.filterPasses==passes+1,'metadata events were not coalesced')
 `);
});

test('sorting is stable, keeps unknown values last, and does not mutate source records or selection',t=>{
 const vm=fixture(t);
 vm.run(`
  P.selected['bags:1']=true
  P.sort='rarity';P.query=nil;P.Refresh()
  assert(P.visible[1].id==3 and P.visible[2].id==1 and P.visible[3].id==2,'rarity order wrong')
  assert(P.visible[#P.visible].quality==nil,'unknown rarity sorted before known values')
  P.sort='level';P.query=nil;P.Refresh()
  assert(P.visible[1].id==1 and P.visible[2].id==2 and P.visible[3].id==3,'level order wrong')
  assert(ITEMS[1].id==1 and P.selected['bags:1'],'sorting mutated source or selection')
  AzerMind.ShowSourceBrowser=function() SOURCE_OPENED=true end
  local row=P.rows[1];row.checked=false;row.scripts.OnClick(row,'RightButton')
  assert(SOURCE_OPENED and row:GetChecked() and P.selected['bags:1'],'source inspection toggled selection')
 `);
});

test('two thousand items page and scroll through bounded rows without rescanning filters',t=>{
 const vm=fixture(t);
 vm.run(`
  LARGE={};for i=1,2000 do LARGE[i]={key='bags:'..i,id=i,name=string.format('Item %04d',i),link='item:'..i,
    count=1,location='Bags',kind='',subtype='Cloth',slot='',classID=4,subclassID=1} end
  AzerMind.OwnedItems=function() return LARGE end;P.Refresh();local passes=P.filterPasses
  assert(P.resultCount:GetText()=='2000 results','matched count unclear')
  P.next.scripts.OnClick();assert(P.rows[1].item.id==26,'next page skipped wrong range')
  P.scroll:SetVerticalScroll(48*10);for _,fn in ipairs(P.scroll.hooks.OnVerticalScroll) do fn() end
  assert(P.rows[1].item.id==36,'scroll did not reach middle of current page')
  assert(#P.rows<=6 and P.filterPasses==passes,'scrolling allocated or rescanned all items')
  assert(P.pageLabel:GetText():find('26-50 / 2000',1,true),'page range incorrect')
 `);
});

test('review includes hidden selections and restores browsing filters and source-specific selections',t=>{
 const vm=fixture(t);
 vm.run(`
  P.selected['bags:1']=true;P.selected['bags:3']=true;P.SelectCategory('sub:4:1')
  assert(#P.visible==1,'browse category wrong')
  P.SetReview(true)
  assert(#P.visible==2 and P.category=='sub:4:1','review discarded hidden selection or filter')
  P.SetReview(false)
  assert(#P.visible==1 and P.visible[1].id==1,'return did not restore filtering')
  AzerMind.CatalogItemList=function() return {{key='catalog:999',id=999,name='Catalog item',link='item:999',count=1,location='Catalog',kind='',subtype='',slot=''}} end
  for _,b in ipairs(P.children) do if b.text=='Item database' then b.scripts.OnClick() end end
  assert(next(P.selected)==nil,'owned selection leaked into catalog')
  P.selected['catalog:999']=true
  for _,b in ipairs(P.children) do if b.text=='My items' then b.scripts.OnClick() end end
  assert(P.selected['bags:1'] and P.selected['bags:3'] and not P.selected['catalog:999'],'source selections were lost or mixed')
 `);
});

test('bulk and individual selection enforce the hundred-item limit atomically',t=>{
 const vm=fixture(t);
 vm.run(`
  LARGE={};for i=1,150 do LARGE[i]={key='bags:'..i,id=i,name=string.format('Item %04d',i),link='item:'..i,
   count=1,location='Bags',kind='',subtype='',slot=''} end
  AzerMind.OwnedItems=function() return LARGE end;P.Refresh()
  P.selectAll.scripts.OnClick();assert(next(P.selected)==nil,'bulk action partially selected too many items')
  for i=1,100 do P.selected['bags:'..i]=true end
  P.Refresh();P.page=5;P.next.scripts.OnClick();P.previous.scripts.OnClick()
  local row=P.rows[1];assert(row.item.id==101,'limit fixture wrong page')
  row.checked=true;row.scripts.OnClick(row,'LeftButton')
  assert(not P.selected['bags:101'] and not row:GetChecked(),'individual limit not enforced')
  assert(not P.selectAll:IsEnabled() and not P.selectPage:IsEnabled(),'oversized bulk action remained enabled')
  assert(AzerMindDB.chats[1].pendingId==nil,'selection sent a message')
 `);
});

test('attachment refresh keeps modifiers, quantities, bank provenance and auction credit without sending',t=>{
 const vm=fixture(t);
 vm.run(`
  ITEMS[1].link='|cff00ff00|Hitem:1:0:0:0:0:0:-19:42|h[Azure Cloth Hat]|h|r'
  ITEMS[1].count=7
  ITEMS[7]={key='bank:1:modified',id=1,name='Bank Hat',link='|Hitem:1:0:0:0:0:0:-20:99|h[Bank Hat]|h',
    count=3,location='Bank',kind='',subtype='Cloth',slot='',classID=4,subclassID=1}
  local key=(GetRealmName() or '')..':'..(UnitName('player') or '')
  AzerMindItemsDB={[key]={updated='2026-11-16 01:00'}}
  AHledgerAPI={GetPrice=function() return {median=12345,market='fixture-market',snapshot=true} end}
  P.selected['bags:1']=true;P.selected['bank:1:modified']=true
  AzerMind.AppendQuestDraft=function(text,chat) ATTACHED=text;ATTACH_CHAT=chat;return true end
  P.insert.scripts.OnClick()
  assert(ATTACHED:find('item:1:0:0:0:0:0:-19:42',1,true),'bag modifiers lost')
  assert(ATTACHED:find('item:1:0:0:0:0:0:-20:99',1,true),'bank variant collapsed into bags')
  assert(ATTACHED:find('×7',1,true) and ATTACHED:find('×3',1,true),'refreshed quantities lost')
  assert(ATTACHED:find('bank snapshot: 2026-11-16 01:00',1,true),'bank age lost')
  assert(ATTACHED:find('Prices from AHledger (ahledger.com)',1,true),'auction provenance lost')
  assert(ATTACH_CHAT==P.chatId and AzerMindDB.chats[1].pendingId==nil,'attachment sent or changed chat')
 `);
});

test('review restores the page and scroll, then reports items removed by an inventory refresh',t=>{
 const vm=fixture(t);
 vm.run(`
  LARGE={};for i=1,80 do LARGE[i]={key='bags:'..i,id=i,name=string.format('Item %04d',i),link='item:'..i,
   count=1,location='Bags',kind='',subtype='',slot=''} end
  AzerMind.OwnedItems=function() return LARGE end;P.Refresh()
  P.selected['bags:1']=true;P.selected['bags:70']=true
  P.next.scripts.OnClick();P.scroll:SetVerticalScroll(144)
  P.SetReview(true);assert(#P.visible==2,'review selection incomplete')
  P.SetReview(false)
  assert(P.page==2 and P.scroll:GetVerticalScroll()==144 and P.rows[1].item.id==29,'browse position was not restored')
  table.remove(LARGE,70);P.Refresh()
  assert(not P.selected['bags:70'] and P.selected['bags:1'] and P.removedSelections==1,'refresh did not reconcile exact keys')
  assert(P.bank:GetText():find('removed',1,true),'unavailable selection not reported')
 `);
});

test('attachments reject changed conversations and oversized payloads without changing the draft',t=>{
 const vm=fixture(t);
 vm.run(`
  P.selected['bags:1']=true
  AzerMind.NewChat();P.insert.scripts.OnClick()
  assert(P.summary:GetText():find('Conversation changed',1,true),'attachment silently changed conversations')
  assert(AzerMindInput:GetText()=='' and AzerMindDB.chats[2].pendingId==nil,'attachment leaked into new conversation')
  AzerMind.ShowItemPicker();P.selected['bags:1']=true
  AzerMind.AuctionItemDetails=function() return string.rep('x',37000) end
  AzerMind.AppendQuestDraft=function() error('oversized draft must not be inserted') end
  P.insert.scripts.OnClick()
  assert(P.summary:GetText():find('too large',1,true) and P:IsShown(),'byte limit did not preserve picker')
 `);
});
